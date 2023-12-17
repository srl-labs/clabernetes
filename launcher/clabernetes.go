package launcher

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"time"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesgeneratedclientset "github.com/srl-labs/clabernetes/generated/clientset"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	"sigs.k8s.io/yaml"
)

const (
	maxDockerLaunchAttempts = 10
	containerCheckInterval  = 5 * time.Second
)

// StartClabernetes is a function that starts the clabernetes launcher. It cannot fail, only panic.
func StartClabernetes() {
	if clabernetesInstance != nil {
		clabernetesutil.Panic("clabernetes instance already created...")
	}

	rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	claberneteslogging.InitManager()

	logManager := claberneteslogging.GetManager()

	clabernetesLogger := logManager.MustRegisterAndGetLogger(
		clabernetesconstants.Clabernetes,
		clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.LauncherLoggerLevelEnv,
			clabernetesconstants.Info,
		),
	)

	containerlabLogger := logManager.MustRegisterAndGetLogger(
		"containerlab",
		clabernetesconstants.Info,
	)

	nodeLogger := logManager.MustRegisterAndGetLogger(
		"node",
		clabernetesconstants.Info,
	)

	ctx, cancel := clabernetesutil.SignalHandledContext(clabernetesLogger.Criticalf)

	clabernetesInstance = &clabernetes{
		ctx:                   ctx,
		cancel:                cancel,
		kubeClabernetesClient: mustNewKubeClabernetesClient(clabernetesLogger),
		appName: clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.AppNameEnv,
			clabernetesconstants.AppNameDefault,
		),
		logger:               clabernetesLogger,
		containerlabLogger:   containerlabLogger,
		nodeLogger:           nodeLogger,
		imageName:            os.Getenv(clabernetesconstants.LauncherNodeImageEnv),
		imagePullThroughMode: os.Getenv(clabernetesconstants.LauncherImagePullThroughModeEnv),
	}

	clabernetesInstance.startup()
}

var clabernetesInstance *clabernetes //nolint:gochecknoglobals

type clabernetes struct {
	ctx    context.Context
	cancel context.CancelFunc

	kubeClabernetesClient *clabernetesgeneratedclientset.Clientset

	appName string

	logger             claberneteslogging.Instance
	containerlabLogger claberneteslogging.Instance
	nodeLogger         claberneteslogging.Instance

	imageName            string
	imagePullThroughMode string

	containerIDs []string
}

func (c *clabernetes) startup() {
	c.logger.Info("starting clabernetes...")

	c.logger.Debugf("clabernetes version %s", clabernetesconstants.Version)

	c.setup()
	c.image()
	c.launch()

	go c.watchContainers()

	c.logger.Info("running for forever or until sigint...")

	<-c.ctx.Done()

	claberneteslogging.GetManager().Flush()
}

func (c *clabernetes) setup() {
	c.logger.Debug("handling mounts...")

	if !strings.EqualFold(
		os.Getenv(clabernetesconstants.LauncherPrivilegedEnv),
		clabernetesconstants.True,
	) {
		c.handleMounts()
	}

	c.logger.Debug("configure insecure registries if requested...")

	err := c.handleInsecureRegistries()
	if err != nil {
		c.logger.Fatalf("failed configuring insecure docker registries, err: %s", err)
	}

	c.logger.Debug("ensuring docker is running...")

	err = c.startDocker()
	if err != nil {
		c.logger.Warn(
			"failed ensuring docker is running, attempting to fallback to legacy ip tables",
		)

		// see https://github.com/srl-labs/clabernetes/issues/47
		err = c.enableLegacyIPTables()
		if err != nil {
			c.logger.Fatalf("failed enabling legacy ip tables, err: %s", err)
		}

		err = c.startDocker()
		if err != nil {
			c.logger.Fatalf("failed ensuring docker is running, err: %s", err)
		}

		c.logger.Warn("docker started, but using legacy ip tables")
	}

	c.logger.Debug("getting files from url if requested...")

	err = c.getFilesFromURL()
	if err != nil {
		c.logger.Fatalf("failed getting file(s) from remote url, err: %s", err)
	}
}

func (c *clabernetes) launch() {
	c.logger.Debug("launching containerlab...")

	err := c.runContainerlab()
	if err != nil {
		c.logger.Fatalf("failed launching containerlab, err: %s", err)
	}

	c.containerIDs = c.getContainerIDs()

	if len(c.containerIDs) > 0 {
		c.logger.Debugf("found container ids %q", c.containerIDs)

		c.tailContainerLogs()
	} else {
		c.logger.Warn(
			"failed determining container ids, will continue but may not be in a working " +
				"state and no container logs will be captured",
		)
	}

	c.logger.Info("containerlab started, setting up any required tunnels...")

	tunnelBytes, err := os.ReadFile("tunnels.yaml")
	if err != nil {
		c.logger.Fatalf("failed loading tunnels yaml file content, err: %s", err)
	}

	var tunnelObj []*clabernetesapisv1alpha1.Tunnel

	err = yaml.Unmarshal(tunnelBytes, &tunnelObj)
	if err != nil {
		c.logger.Fatalf("failed unmarshalling tunnels config, err: %s", err)
	}

	for _, tunnel := range tunnelObj {
		err = c.runContainerlabVxlanTools(
			tunnel.LocalNodeName,
			tunnel.LocalLinkName,
			tunnel.RemoteName,
			tunnel.ID,
		)
		if err != nil {
			c.logger.Fatalf(
				"failed setting up tunnel to remote node '%s' for local interface '%s', error: %s",
				tunnel.RemoteNodeName,
				tunnel.LocalLinkName,
				err,
			)
		}
	}
}

func (c *clabernetes) watchContainers() {
	if len(c.containerIDs) == 0 {
		return
	}

	ticker := time.NewTicker(containerCheckInterval)

	for range ticker.C {
		currentContainerIDs := c.getContainerIDs()

		if len(currentContainerIDs) != len(c.containerIDs) {
			c.logger.Criticalf(
				"expected %d running containers, but got %d, sending done signal",
				len(c.containerIDs),
				len(currentContainerIDs),
			)

			c.cancel()

			return
		}
	}
}

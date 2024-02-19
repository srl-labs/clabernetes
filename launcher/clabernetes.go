package launcher

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesgeneratedclientset "github.com/srl-labs/clabernetes/generated/clientset"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

const (
	maxDockerLaunchAttempts = 10
	containerCheckInterval  = 5 * time.Second
	clientDefaultTimeout    = time.Minute
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

	c.containerlabVersion()
	c.setup()
	c.image()
	c.launch()
	c.connectivity()

	go c.watchContainers()

	c.logger.Info("running for forever or until sigint...")

	<-c.ctx.Done()

	claberneteslogging.GetManager().Flush()
}

func (c *clabernetes) containerlabVersion() {
	c.logger.Debug("checking containerlab version settings...")

	requestedVersion := os.Getenv(clabernetesconstants.LauncherContainerlabVersion)

	if requestedVersion == "" {
		c.logger.Debug("no custom containerlab version specified, continuing....")

		return
	}

	err := c.installContainerlabVersion(requestedVersion)
	if err != nil {
		c.logger.Fatalf("failed installing requested containerlab version, err: %s", err)
	}

	c.logger.Debug("requested containerlab version installed successfully")
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

	c.logger.Debug("containerlab launched successfully")
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

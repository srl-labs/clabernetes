package launcher

import (
	"context"
	"math/rand"
	"os"
	"time"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	"sigs.k8s.io/yaml"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

const (
	maxDockerLaunchAttempts = 10
)

// StartClabernetes is a function that starts the clabernetes launcher.
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

	ctx, _ := clabernetesutil.SignalHandledContext(clabernetesLogger.Criticalf)

	clabernetesInstance = &clabernetes{
		ctx: ctx,
		appName: clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.AppNameEnvVar,
			clabernetesconstants.AppNameDefault,
		),
		logger:             clabernetesLogger,
		containerlabLogger: containerlabLogger,
		nodeLogger:         nodeLogger,
	}

	clabernetesInstance.startup()
}

var clabernetesInstance *clabernetes //nolint:gochecknoglobals

type clabernetes struct {
	ctx context.Context

	appName string

	logger             claberneteslogging.Instance
	containerlabLogger claberneteslogging.Instance
	nodeLogger         claberneteslogging.Instance
}

func (c *clabernetes) startup() {
	c.logger.Info("starting clabernetes...")

	c.logger.Debugf("clabernetes version %s", clabernetesconstants.Version)

	c.setup()
	c.launch()

	c.logger.Info("running for forever or until sigint...")

	<-c.ctx.Done()
}

func (c *clabernetes) setup() {
	c.logger.Debug("configure insecure registries if requested...")

	err := c.handleInsecureRegistries()
	if err != nil {
		c.logger.Criticalf("failed configuring insecure docker registries, err: %s", err)

		clabernetesutil.Panic(err.Error())
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
			c.logger.Criticalf("failed enabling legacy ip tables, err: %s", err)

			clabernetesutil.Panic(err.Error())
		}

		err = c.startDocker()
		if err != nil {
			c.logger.Criticalf("failed ensuring docker is running, err: %s", err)

			clabernetesutil.Panic(err.Error())
		}

		c.logger.Warn("docker started, but using legacy ip tables")
	}
}

func (c *clabernetes) launch() {
	c.logger.Debug("launching containerlab...")

	err := c.runContainerlab()
	if err != nil {
		c.logger.Criticalf("failed launching containerlab, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	containerIDs := c.getContainerIDs()
	c.tailContainerLogs(containerIDs)

	c.logger.Info("containerlab started, setting up any required tunnels...")

	tunnelBytes, err := os.ReadFile("tunnels.yaml")
	if err != nil {
		c.logger.Criticalf("failed loading tunnels yaml file content, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	var tunnelObj []*clabernetesapistopologyv1alpha1.Tunnel

	err = yaml.Unmarshal(tunnelBytes, &tunnelObj)
	if err != nil {
		c.logger.Criticalf("failed unmarshalling tunnels config, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	for _, tunnel := range tunnelObj {
		err = c.runContainerlabVxlanTools(
			tunnel.LocalNodeName,
			tunnel.LocalLinkName,
			tunnel.RemoteName,
			tunnel.ID,
		)
		if err != nil {
			c.logger.Criticalf(
				"failed setting up tunnel to remote node '%s' for local interface '%s', error: %s",
				tunnel.RemoteNodeName,
				tunnel.LocalLinkName,
				err,
			)

			clabernetesutil.Panic(err.Error())
		}
	}
}

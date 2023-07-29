package launcher

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"time"

	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"

	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"
	"gopkg.in/yaml.v3"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	claberneteslogging "gitlab.com/carlmontanari/clabernetes/logging"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
)

const maxDockerLaunchAttempts = 10

// StartClabernetes is a function that starts the clabernetes launcher.
func StartClabernetes() {
	if clabernetesInstance != nil {
		clabernetesutil.Panic("clabernetes instance already created...")
	}

	rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	logManager := claberneteslogging.GetManager()

	clabernetesLogger := logManager.MustRegisterAndGetLogger(
		clabernetesconstants.Clabernetes,
		clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.LauncherLoggerLevelEnv,
			clabernetesconstants.Info,
		),
	)

	clabernetesInstance = &clabernetes{
		ctx: clabernetesutil.SignalHandledContext(clabernetesLogger.Criticalf),
		appName: clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.AppNameEnvVar,
			clabernetesconstants.AppNameDefault,
		),
		logger: clabernetesLogger,
	}

	clabernetesInstance.startup()
}

var clabernetesInstance *clabernetes //nolint:gochecknoglobals

type clabernetes struct {
	ctx context.Context

	appName string

	logger claberneteslogging.Instance
}

func (c *clabernetes) startup() {
	c.logger.Info("starting clabernetes...")

	c.logger.Debug("ensuring docker is running...")

	err := c.startDocker()
	if err != nil {
		c.logger.Criticalf("failed ensuring docker is running, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	c.logger.Debug("launching containerlab...")

	err = c.runClab()
	if err != nil {
		c.logger.Criticalf("failed launching containerlab, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	c.logger.Info("containerlab started, setting up any required tunnels...")

	tunnelBytes, err := os.ReadFile("tunnels.yaml")
	if err != nil {
		c.logger.Criticalf("failed loading tunnels yaml file content, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	var tunnelObj []*clabernetesapistopology.Tunnel

	err = yaml.Unmarshal(tunnelBytes, &tunnelObj)
	if err != nil {
		c.logger.Criticalf("failed unmarshalling tunnels config, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	for _, tunnel := range tunnelObj {
		err = c.runClabVxlanTools(
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

	c.logger.Info("running for forever or until sigint...")

	<-c.ctx.Done()
}

func (c *clabernetes) startDocker() error {
	// this is janky, why am i too dumb to make docker start in the container?! (no systemd things)
	var attempts int

	for {
		psCmd := exec.Command("docker", "ps")

		_, err := psCmd.Output()
		if err == nil {
			// exit 0, docker seems happy
			return nil
		}

		if attempts > maxDockerLaunchAttempts {
			return fmt.Errorf("%w: failed starting docker", claberneteserrors.ErrLaunch)
		}

		cmd := exec.Command("service", "docker", "start")

		_, err = cmd.Output()
		if err != nil {
			return err
		}

		time.Sleep(time.Second)

		attempts++
	}
}

func (c *clabernetes) runClab() error {
	cmd := exec.Command("containerlab", "deploy", "-t", "topo.yaml", "--debug")

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	c.logger.Debugf("containerlab start output:\n%s", string(output))

	return nil
}

func (c *clabernetes) runClabVxlanTools(
	localNodeName, cntLink, vxlanRemote string,
	vxlanID int,
) error {
	resolvedVxlanRemotes, err := net.LookupIP(vxlanRemote)
	if err != nil {
		return err
	}

	if len(resolvedVxlanRemotes) != 1 {
		return fmt.Errorf(
			"%w: did not get exactly one ip resolved for remote vxlan endpoint",
			claberneteserrors.ErrConnectivity,
		)
	}

	resolvedVxlanRemote := resolvedVxlanRemotes[0].String()

	cmd := exec.Command( //nolint:gosec
		"containerlab",
		"tools",
		"vxlan",
		"create",
		"--remote",
		resolvedVxlanRemote,
		"--id",
		fmt.Sprint(vxlanID),
		"--link",
		fmt.Sprintf("%s-%s", localNodeName, cntLink),
	)

	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

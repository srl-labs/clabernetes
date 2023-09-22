package launcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	claberneteserrors "github.com/srl-labs/clabernetes/errors"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	"gopkg.in/yaml.v3"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
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

	c.logger.Debug("configure insecure registries if requested...")

	err := c.handleInsecureRegistries()
	if err != nil {
		c.logger.Criticalf("failed configuring insecure docker registries, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	c.logger.Debug("ensuring docker is running...")

	err = c.startDocker()
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

	var tunnelObj []*clabernetesapistopologyv1alpha1.Tunnel

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

func (c *clabernetes) handleInsecureRegistries() error {
	insecureRegistries := os.Getenv(clabernetesconstants.LauncherInsecureRegistries)

	if insecureRegistries == "" {
		return nil
	}

	splitRegistries := strings.Split(insecureRegistries, ",")

	quotedRegistries := make([]string, len(splitRegistries))

	for idx, elem := range splitRegistries {
		quotedRegistries[idx] = fmt.Sprintf("%q", elem)
	}

	templateVars := struct {
		InsecureRegistries string
	}{
		InsecureRegistries: strings.Join(quotedRegistries, ","),
	}

	t, err := template.ParseFS(Assets, "assets/docker-daemon.json.template")
	if err != nil {
		return err
	}

	var rendered bytes.Buffer

	err = t.Execute(&rendered, templateVars)
	if err != nil {
		return err
	}

	err = os.WriteFile(
		"/etc/docker/daemon.json",
		rendered.Bytes(),
		clabernetesconstants.PermissionsEveryoneRead,
	)
	if err != nil {
		return err
	}

	return nil
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
	clabLogFile, err := os.Create("clab.log")
	if err != nil {
		return err
	}

	clabOutWriter := io.MultiWriter(c.logger, clabLogFile)

	args := []string{
		"deploy",
		"-t",
		"topo.yaml",
	}

	if os.Getenv(clabernetesconstants.LauncherContainerlabDebug) == clabernetesconstants.True {
		args = append(args, "--debug")
	}

	cmd := exec.Command("containerlab", args...)

	cmd.Stdout = clabOutWriter
	cmd.Stderr = clabOutWriter

	err = cmd.Run()
	if err != nil {
		return err
	}

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

	c.logger.Debugf("resolved remote vxlan tunnel service address as '%s'", resolvedVxlanRemote)

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

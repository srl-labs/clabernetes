package launcher

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
)

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

func (c *clabernetes) enableLegacyIPTables() error {
	updateCmd := exec.Command(
		"update-alternatives",
		"--set",
		"iptables",
		"/usr/sbin/iptables-legacy",
	)

	updateCmd.Stdout = c.logger
	updateCmd.Stderr = c.logger

	err := updateCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *clabernetes) startDocker() error {
	var attempts int

	for {
		psCmd := exec.Command("docker", "ps")

		psCmd.Stdout = c.logger
		psCmd.Stderr = c.logger

		err := psCmd.Run()
		if err == nil {
			// exit 0, docker seems happy
			return nil
		}

		if attempts > maxDockerLaunchAttempts {
			return fmt.Errorf("%w: failed starting docker", claberneteserrors.ErrLaunch)
		}

		startCmd := exec.Command("service", "docker", "start")

		startCmd.Stdout = c.logger
		startCmd.Stderr = c.logger

		err = startCmd.Run()
		if err != nil {
			return err
		}

		time.Sleep(time.Second)

		attempts++
	}
}

func (c *clabernetes) getContainerIDs() []string {
	// return all the container ids running in the pod
	psCmd := exec.Command("docker", "ps", "--quiet")

	output, err := psCmd.Output()
	if err != nil {
		c.logger.Warnf(
			"failed determining container ids will continue but will not log container output,"+
				" err: %s",
			err,
		)

		return nil
	}

	containerIDLines := strings.Split(string(output), "\n")

	var containerIDs []string

	for _, line := range containerIDLines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine != "" {
			containerIDs = append(containerIDs, trimmedLine)
		}
	}

	return containerIDs
}

func (c *clabernetes) tailContainerLogs() {
	nodeLogFile, err := os.Create("node.log")
	if err != nil {
		c.logger.Warnf("failed creating node log file, err: %s", err)

		return
	}

	nodeOutWriter := io.MultiWriter(c.nodeLogger, nodeLogFile)

	for _, containerID := range c.containerIDs {
		go func(containerID string, nodeOutWriter io.Writer) {
			args := []string{
				"logs",
				"-f",
				containerID,
			}

			cmd := exec.Command("docker", args...) //nolint:gosec

			cmd.Stdout = nodeOutWriter
			cmd.Stderr = nodeOutWriter

			err = cmd.Run()
			if err != nil {
				c.logger.Warnf(
					"tailing node logs for container id %q failed, err: %s", containerID, err,
				)
			}
		}(containerID, nodeOutWriter)
	}
}

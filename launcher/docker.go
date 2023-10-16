package launcher

import (
	"bytes"
	"fmt"
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

	_, err := updateCmd.Output()
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

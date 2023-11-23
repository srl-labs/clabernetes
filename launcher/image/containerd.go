package image

import (
	"fmt"
	"os/exec"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

type containerdManager struct {
	logger claberneteslogging.Instance
}

func (m *containerdManager) Present(imageName string) (bool, error) {
	checkCmd := exec.Command( //nolint:gosec
		"nerdctl",
		"--address",
		"/clabernetes/.node/containerd.sock",
		"--namespace",
		"k8s.io",
		"image",
		"list",
		"--filter",
		fmt.Sprintf("reference=%s", imageName),
		"--quiet",
	)

	output, err := checkCmd.Output()
	if err != nil {
		return false, err
	}

	if len(output) == 0 {
		return false, nil
	}

	return true, nil
}

func (m *containerdManager) Pull(imageName string) error {
	pullCmd := exec.Command(
		"nerdctl",
		"--address",
		"/clabernetes/.node/containerd.sock",
		"--namespace",
		"k8s.io",
		"image",
		"pull",
		imageName,
		"--quiet",
	)

	pullCmd.Stdout = m.logger
	pullCmd.Stderr = m.logger

	err := pullCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (m *containerdManager) Export(imageName, destination string) error {
	exportCmd := exec.Command(
		"nerdctl",
		"--address",
		"/clabernetes/.node/containerd.sock",
		"--namespace",
		"k8s.io",
		"image",
		"save",
		"--output",
		destination,
		imageName,
	)

	exportCmd.Stdout = m.logger
	exportCmd.Stderr = m.logger

	err := exportCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

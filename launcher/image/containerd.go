package image

import (
	"os/exec"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

type containerdManager struct {
	logger claberneteslogging.Instance
}

func (m *containerdManager) Present(imageName string) (bool, error) {
	_ = imageName

	return false, nil
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
		"-o",
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

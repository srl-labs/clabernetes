package launcher

import (
	"fmt"
	"os"
	"os/exec"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func (c *clabernetes) image() {
	imagePullThroughMode := os.Getenv(clabernetesconstants.LauncherImagePullThroughModeEnv)

	switch imagePullThroughMode {
	case clabernetesconstants.ImagePullThroughModeAuto,
		clabernetesconstants.ImagePullThroughModeAlways:
		c.logger.Infof(
			"unknown image pull through mode %q, start image pull through attempt...",
			imagePullThroughMode,
		)
	case clabernetesconstants.ImagePullThroughModeNever:
		c.logger.Debugf(
			"image pull through mode is %q, skipping image pull through...",
			imagePullThroughMode,
		)

		return
	default:
		c.logger.Warnf(
			"unknown image pull through mode %q, skipping image pull through...",
			imagePullThroughMode,
		)

		return
	}

	c.logger.Debug("handling image pull through...")

	criKind := os.Getenv(clabernetesconstants.LauncherCRIKindEnv)

	if criKind == "" || criKind == clabernetesconstants.KubernetesCRIUnknown {
		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			msg := fmt.Sprintf(
				"image pull through mode is always, but criKind is unset or unknown," +
					" cannot continue...",
			)

			c.logger.Critical(msg)

			clabernetesutil.Panic(msg)
		}

		c.logger.Warn(
			"image pull through mode is auto, but criKind is not set or unknown," +
				" continuing to normal launch...",
		)

		return
	}

	imageName := os.Getenv(clabernetesconstants.LauncherNodeImageEnv)
	if imageName == "" {
		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			msg := fmt.Sprintf(
				"image pull through mode is always, node image is unknown," +
					" cannot continue...",
			)

			c.logger.Critical(msg)

			clabernetesutil.Panic(msg)
		}

		c.logger.Warn(
			"image pull through mode is auto, but node image is unknown," +
				" continuing to normal launch...",
		)

		return
	}

	var err error

	switch criKind {
	case clabernetesconstants.KubernetesCRIContainerd:
		c.logger.Info("attempting containerd image pull through...")

		err = c.imageContainerd(imageName)
	default:
		clabernetesutil.Panic(
			"image pull through not implemented for anything but containerd, this is a bug",
		)
	}

	if err != nil {
		c.logger.Warnf("failed image pull through (pull), err: %s", err)

		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			clabernetesutil.Panic(
				"image pull through failed and pull through mode is always, cannot continue",
			)
		}

		// if mode is *not* always we can fail through to try to let docker handle it
		return
	}

	err = c.imageImport()
	if err != nil {
		c.logger.Warnf("failed image pull through (import), err: %s", err)

		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			clabernetesutil.Panic(
				"image pull through failed and pull through mode is always, cannot continue",
			)
		}
	}
}

func (c *clabernetes) imageContainerd(imageName string) error {
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

	pullCmd.Stdout = c.logger
	pullCmd.Stderr = c.logger

	err := pullCmd.Run()
	if err != nil {
		return err
	}

	exportCmd := exec.Command(
		"nerdctl",
		"--address",
		"/clabernetes/.node/containerd.sock",
		"--namespace",
		"k8s.io",
		"image",
		"save",
		"-o",
		"/clabernetes/.image/node-image.tar",
		imageName,
	)

	exportCmd.Stdout = c.logger
	exportCmd.Stderr = c.logger

	err = exportCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *clabernetes) imageImport() error {
	exportCmd := exec.Command(
		"docker",
		"image",
		"load",
		"-i",
		"/clabernetes/.image/node-image.tar",
	)

	exportCmd.Stdout = c.logger
	exportCmd.Stderr = c.logger

	err := exportCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

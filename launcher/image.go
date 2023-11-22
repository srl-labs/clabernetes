package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	claberneteserrors "github.com/srl-labs/clabernetes/errors"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslauncherimage "github.com/srl-labs/clabernetes/launcher/image"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

const imageDestination = "/clabernetes/.image/node-image.tar"

func (c *clabernetes) image() {
	imagePullThroughMode := os.Getenv(clabernetesconstants.LauncherImagePullThroughModeEnv)

	switch imagePullThroughMode {
	case clabernetesconstants.ImagePullThroughModeAuto,
		clabernetesconstants.ImagePullThroughModeAlways:
		c.logger.Infof(
			"image pull through mode %q, start image pull through attempt...",
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

	imageManager, err := claberneteslauncherimage.NewImageManager(
		c.logger,
		os.Getenv(clabernetesconstants.LauncherCRIKindEnv),
	)
	if err != nil {
		c.logger.Warnf("error creating image manager, err: %s", err)

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

	imagePresent, err := imageManager.Present(imageName)
	c.handleImagePullThroughError(err, imagePullThroughMode, "check")

	if imagePresent {
		c.logger.Infof("image %q is present, aborting image pull through", imageName)

		return
	}

	// TODO -- here we send a request to manager to run a job that spawns a pod on this node

	err = c.waitForImage(imageName, imageManager)
	c.handleImagePullThroughError(err, imagePullThroughMode, "wait")

	err = imageManager.Pull(imageName)
	c.handleImagePullThroughError(err, imagePullThroughMode, "pull")

	err = imageManager.Export(imageName, imageDestination)
	c.handleImagePullThroughError(err, imagePullThroughMode, "export")

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

func (c *clabernetes) handleImagePullThroughError(
	err error,
	imagePullThroughMode, imagePullThroughStage string,
) {
	if err != nil {
		c.logger.Warnf("failed image pull through (%s), err: %s", imagePullThroughStage, err)

		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			clabernetesutil.Panic(
				"image pull through failed and pull through mode is always, cannot continue",
			)
		}
	}
}

func (c *clabernetes) waitForImage(
	imageName string,
	imageManager claberneteslauncherimage.Manager,
) error {
	startTime := time.Now()

	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		if time.Since(startTime) > 5*time.Minute {
			break
		}

		imagePresent, err := imageManager.Present(imageName)
		if err != nil {
			return err
		}

		if imagePresent {
			return nil
		}
	}

	return fmt.Errorf(
		"%w: timed out waiting for image %q to be present on node",
		claberneteserrors.ErrLaunch,
		imageName,
	)
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

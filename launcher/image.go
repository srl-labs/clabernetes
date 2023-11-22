package launcher

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	claberneteshttptypes "github.com/srl-labs/clabernetes/http/types"

	claberneteserrors "github.com/srl-labs/clabernetes/errors"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslauncherimage "github.com/srl-labs/clabernetes/launcher/image"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

const (
	imagePullRequestTimeout = 5 * time.Second
	imageDestination        = "/clabernetes/.image/node-image.tar"
	imageCheckPollInterval  = 5 * time.Second
	imageCheckLogCounter    = 6
)

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

	continuePullThrough := c.imageCheckPresent(imageManager, imageName, imagePullThroughMode)
	if !continuePullThrough {
		return
	}

	err = c.requestImagePull(imageName)
	if err != nil {
		c.logger.Warnf("failed image pull through (request pull), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(imagePullThroughMode)

		return
	}

	err = c.waitForImage(imageName, imageManager)
	if err != nil {
		c.logger.Warnf("failed image pull through (wait), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(imagePullThroughMode)

		return
	}

	err = imageManager.Export(imageName, imageDestination)
	if err != nil {
		c.logger.Warnf("failed image pull through (export), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(imagePullThroughMode)

		return
	}

	err = c.imageImport()
	if err != nil {
		c.logger.Warnf("failed image pull through (import), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(imagePullThroughMode)
	}
}

func handleImagePullThroughModeAlwaysPanic(imagePullThroughMode string) {
	if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
		clabernetesutil.Panic(
			"image pull through failed and pull through mode is always, cannot continue",
		)
	}
}

func (c *clabernetes) requestImagePull(imageName string) error {
	imageRequest := claberneteshttptypes.ImageRequest{
		TopologyName:       os.Getenv(clabernetesconstants.LauncherTopologyNameEnv),
		TopologyNamespace:  os.Getenv(clabernetesconstants.PodNamespaceEnv),
		TopologyNodeName:   os.Getenv(clabernetesconstants.LauncherNodeNameEnv),
		KubernetesNodeName: os.Getenv(clabernetesconstants.NodeNameEnv),
		RequestingPodName:  os.Getenv(clabernetesconstants.PodNameEnv),
		RequestedImageName: imageName,
	}

	requestJSON, err := json.Marshal(imageRequest)
	if err != nil {
		c.logger.Criticalf("failed marshaling image pull request, error: %s", err)

		return err
	}

	body := bytes.NewReader(requestJSON)

	ctx, cancel := context.WithTimeout(c.ctx, imagePullRequestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf(
			"https://%s.%s/image",
			fmt.Sprintf("%s-http", os.Getenv(clabernetesconstants.AppNameEnv)),
			os.Getenv(clabernetesconstants.ManagerNamespaceEnv),
		),
		body,
	)
	if err != nil {
		c.logger.Criticalf("failed building image pull request, error: %s", err)

		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}}

	response, err := client.Do(request)
	if err != nil {
		c.logger.Criticalf("failed executing image pull request, error: %s", err)

		return err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		c.logger.Criticalf("failed reading image pull request response, error: %s", err)

		return err
	}

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf(
			"received non 200 status code from image pull request, response body: %s",
			string(responseBody),
		)

		c.logger.Criticalf(msg)

		return fmt.Errorf("%w: %s", claberneteserrors.ErrLaunch, msg)
	}

	_ = response.Body.Close()

	return nil
}

// return value is "should we continue (true) or with pull through process or not".
func (c *clabernetes) imageCheckPresent(
	imageManager claberneteslauncherimage.Manager,
	imageName, imagePullThroughMode string,
) bool {
	imagePresent, err := imageManager.Present(imageName)
	if err != nil {
		c.logger.Warnf("failed image pull through (check), err: %s", err)

		if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			clabernetesutil.Panic(
				"image pull through failed and pull through mode is always, cannot continue",
			)
		}

		return false
	}

	if imagePresent {
		c.logger.Infof("image %q is present, aborting image pull through", imageName)

		return false
	}

	return true
}

func (c *clabernetes) waitForImage(
	imageName string,
	imageManager claberneteslauncherimage.Manager,
) error {
	startTime := time.Now()

	ticker := time.NewTicker(imageCheckPollInterval)

	var checkCounter int

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

		checkCounter++

		if checkCounter == imageCheckLogCounter {
			checkCounter = 0

			c.logger.Infof("waiting for image %q to be present on node...", imageName)
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

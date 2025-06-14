package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
    "strings"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslauncherimage "github.com/srl-labs/clabernetes/launcher/image"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
	"gopkg.in/yaml.v3"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	imageDestination       = "/clabernetes/.node/shared-volume/"
	imageCheckPollInterval = 5 * time.Second
	imageCheckLogCounter   = 6
)

func generateImageRequestCRName(nodeName, imageName string) string {
	// hash the image name so it doesn't contain invalid chars for k8s name
	return clabernetesutilkubernetes.SafeConcatNameKubernetes(
		nodeName, clabernetesutil.HashBytes([]byte(imageName)),
	)
}

func (c *clabernetes) image() {
	abort, imageManager := c.prepareImagePullThrough()
	if abort {
		return
	}

	imagePresent, err := imageManager.Present(c.imageName)
	if err != nil {
		c.logger.Warnf("failed image pull through (check), err: %s", err)

		if c.imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			c.logger.Fatal(
				"image pull through failed and pull through mode is always, cannot continue",
			)
		}

		c.logger.Warnf("attempting to continue without image pull through...")

		return
	}

	if imagePresent {
		c.logger.Infof("image %q is present, begin copy to docker daemon...", c.imageName)

		c.copyImageFromCRI(imageManager)

		return
	}

	configuredPullSecretsBytes, err := os.ReadFile("configured-pull-secrets.yaml")
	if err != nil {
		c.logger.Warnf("failed image pull through (read secrets), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(c.imagePullThroughMode)

		return
	}

	var configuredPullSecrets []string

	err = yaml.Unmarshal(configuredPullSecretsBytes, &configuredPullSecrets)
	if err != nil {
		c.logger.Warnf("failed image pull through (unmarshal secrets), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(c.imagePullThroughMode)

		return
	}

	err = c.requestImagePull(imageManager, configuredPullSecrets)
	if err != nil {
		handleImagePullThroughModeAlwaysPanic(c.imagePullThroughMode)

		return
	}

	c.copyImageFromCRI(imageManager)
}

func (c *clabernetes) copyImageFromCRI(imageManager claberneteslauncherimage.Manager) {
	err := imageManager.Export(c.imageName, imageDestination)
	if err != nil {
		c.logger.Warnf("failed image pull through (export), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(c.imagePullThroughMode)

		return
	}

	err = c.imageImport(c.imageName, imageDestination)
	if err != nil {
		c.logger.Warnf("failed image pull through (import), err: %s", err)

		handleImagePullThroughModeAlwaysPanic(c.imagePullThroughMode)
	}
}

func (c *clabernetes) prepareImagePullThrough() (
	abort bool,
	imageManager claberneteslauncherimage.Manager,
) {
	switch c.imagePullThroughMode {
	case clabernetesconstants.ImagePullThroughModeAuto,
		clabernetesconstants.ImagePullThroughModeAlways:
		c.logger.Infof(
			"image pull through mode %q, start image pull through attempt...",
			c.imagePullThroughMode,
		)
	case clabernetesconstants.ImagePullThroughModeNever:
		c.logger.Debugf(
			"image pull through mode is %q, skipping image pull through...",
			c.imagePullThroughMode,
		)

		return true, nil
	default:
		c.logger.Warnf(
			"unknown image pull through mode %q, skipping image pull through...",
			c.imagePullThroughMode,
		)

		return true, nil
	}

	c.logger.Debug("handling image pull through...")

	imageManager, err := claberneteslauncherimage.NewManager(
		c.logger,
		os.Getenv(clabernetesconstants.LauncherCRIKindEnv),
	)
	if err != nil {
		c.logger.Warnf("error creating image manager, err: %s", err)

		if c.imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			c.logger.Fatal("image pull through mode is always, but criKind is unset or unknown," +
				" cannot continue...")
		}

		c.logger.Warn(
			"image pull through mode is auto, but criKind is not set or unknown," +
				" continuing to normal launch...",
		)

		return true, nil
	}

	if c.imageName == "" {
		if c.imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
			c.logger.Fatal(
				"image pull through mode is always, node image is unknown," +
					" cannot continue...",
			)
		}

		c.logger.Warn(
			"image pull through mode is auto, but node image is unknown," +
				" continuing to normal launch...",
		)

		return true, nil
	}

	return false, imageManager
}

func handleImagePullThroughModeAlwaysPanic(imagePullThroughMode string) {
	if imagePullThroughMode == clabernetesconstants.ImagePullThroughModeAlways {
		clabernetesutil.Panic(
			"image pull through failed and pull through mode is always, cannot continue",
		)
	}
}

func (c *clabernetes) requestImagePull(
	imageManager claberneteslauncherimage.Manager,
	configuredPullSecrets []string,
) error {
	nodeName := os.Getenv(clabernetesconstants.NodeNameEnv)

	imageRequestCRName := generateImageRequestCRName(nodeName, c.imageName)

	err := c.createImageRequestCR(nodeName, imageRequestCRName, configuredPullSecrets)
	if err != nil {
		c.logger.Warnf("failed image pull through (create request), err: %s", err)

		return err
	}

	err = c.waitImageRequestCRAccepted(imageRequestCRName)
	if err != nil {
		c.logger.Warnf("failed image pull through (wait accepted), err: %s", err)

		return err
	}

	err = c.waitForImage(imageManager)
	if err != nil {
		c.logger.Warnf("failed image pull through (wait image present), err: %s", err)

		return err
	}

	return nil
}

func (c *clabernetes) createImageRequestCR(
	nodeName, imageRequestCRName string,
	configuredPullSecrets []string,
) error {
	ctx, cancel := context.WithTimeout(c.ctx, clientDefaultTimeout)
	defer cancel()

	_, err := c.kubeClabernetesClient.ClabernetesV1alpha1().
		ImageRequests(os.Getenv(clabernetesconstants.PodNamespaceEnv)).
		Create(
			ctx,
			&clabernetesapisv1alpha1.ImageRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: imageRequestCRName,
				},
				Spec: clabernetesapisv1alpha1.ImageRequestSpec{
					TopologyName: os.Getenv(
						clabernetesconstants.LauncherTopologyNameEnv,
					),
					TopologyNodeName:          os.Getenv(clabernetesconstants.LauncherNodeNameEnv),
					KubernetesNode:            nodeName,
					RequestedImage:            c.imageName,
					RequestedImagePullSecrets: configuredPullSecrets,
				},
			},
			metav1.CreateOptions{},
		)
	if err != nil {
		if apimachineryerrors.IsAlreadyExists(err) {
			// if it already exists some other launcher has requested this image for this node
			return nil
		}

		// any other error would be a bad bingo
		return err
	}

	return nil
}

func (c *clabernetes) waitImageRequestCRAccepted(imageRequestCRName string) error {
	startTime := time.Now()

	ticker := time.NewTicker(imageCheckPollInterval)

	for range ticker.C {
		if time.Since(startTime) > clabernetesconstants.PullerPodTimeout {
			break
		}

		ctx, cancel := context.WithTimeout(c.ctx, clientDefaultTimeout)

		imageRequestCR, err := c.kubeClabernetesClient.ClabernetesV1alpha1().
			ImageRequests(os.Getenv(clabernetesconstants.PodNamespaceEnv)).
			Get(
				ctx,
				imageRequestCRName,
				metav1.GetOptions{},
			)

		cancel()

		if err != nil {
			return err
		}

		if imageRequestCR.Status.Accepted {
			// cr has been "accepted" meaning controller will handle getting the image pulled on
			// our node.
			return nil
		}
	}

	return fmt.Errorf(
		"%w: timed out waiting for image request cr %q to change to accepted state",
		claberneteserrors.ErrLaunch,
		imageRequestCRName,
	)
}

func (c *clabernetes) waitForImage(
	imageManager claberneteslauncherimage.Manager,
) error {
	startTime := time.Now()

	ticker := time.NewTicker(imageCheckPollInterval)

	var checkCounter int

	for range ticker.C {
		if time.Since(startTime) > clabernetesconstants.PullerPodTimeout {
			break
		}

		imagePresent, err := imageManager.Present(c.imageName)
		if err != nil {
			return err
		}

		if imagePresent {
			c.logger.Infof("image %q is now available on node, continuing...", c.imageName)

			return nil
		}

		checkCounter++

		if checkCounter == imageCheckLogCounter {
			checkCounter = 0

			c.logger.Infof("waiting for image %q to be present on node...", c.imageName)
		}
	}

	return fmt.Errorf(
		"%w: timed out waiting for image %q to be present on node",
		claberneteserrors.ErrLaunch,
		c.imageName,
	)
}

func (c *clabernetes) imageImport(imageName, destination string) error {
    safeName := imageName
    if idx := strings.LastIndex(imageName, "/"); idx != -1 {
        safeName = imageName[idx+1:]
    }
    // Replace : with _
    safeName = strings.ReplaceAll(safeName, ":", "_")
	lockPath := destination + safeName + ".lock"
	tarPath := destination + safeName + ".tar"
    // Let's measure and log elapsed time
    start := time.Now()
    defer func() {
        elapsed := time.Since(start)
        c.logger.Infof("Import of image %q took %s", imageName, elapsed)
    }()
	// Wait 3 sec until the export lock file is gone (export finished) just in case
	for {
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			break
		}
		time.Sleep(3 * time.Second)
	}

    c.logger.Infof("Lock file is gone for %q, starting Docker import..", c.imageName)
	importCmd := exec.Command(
		"docker",
		"image",
		"load",
		"-i",
		tarPath,
	)

	importCmd.Stdout = c.logger
	importCmd.Stderr = c.logger

	if err := importCmd.Run(); err != nil {
		return err
	}
    c.logger.Infof("Docker import %q completed!", c.imageName)
	return nil
}

func (c *clabernetes) imageCleanup() {
	c.logger.Debug("running image (docker) cleanup in background...")

	exportCmd := exec.Command(
		"docker",
		"system",
		"prune",
		"--force",
	)

	exportCmd.Stdout = c.logger
	exportCmd.Stderr = c.logger

	err := exportCmd.Run()
	if err != nil {
		c.logger.Warnf("failed pruning docker daemon, error: %s", err)
	}
}

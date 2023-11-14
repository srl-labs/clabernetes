package launcher

import (
	"context"
	"fmt"
	"os"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"

	containerdclient "github.com/containerd/containerd/v2/client"
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

	// TODO -- need to load clab topo to get the image here so we know what to load/export/import

	switch criKind {
	case clabernetesconstants.KubernetesCRIContainerd:
		c.logger.Info("attempting containerd image pull through...")

		c.imageContainerd()
	default:
		clabernetesutil.Panic(
			"image pull through not implemented for anything but containerd, this is a bug",
		)
	}
}

func (c *clabernetes) imageContainerd() {
	// TODO -- yolo, just shell out cuz we already have ctr and then we have less deps which is life
	client, err := containerdclient.New(
		fmt.Sprintf(
			"%s/%s",
			clabernetesconstants.LauncherCRISockPath,
			clabernetesconstants.KubernetesCRISockContainerd,
		),
	)
	if err != nil {

	}

	baseCtx := context.Background()

	imagePullDeadline := time.Now().Add(5 * time.Minute)

	pullCtx, pullCtxCancel := context.WithDeadline(baseCtx, imagePullDeadline)

	imageName := "docker.io/hello-world:latest"

	iamge, err := client.Pull(pullCtx, imageName)
	pullCtxCancel()

	if err != nil {

	}

	exportCtx, exportCtxCancel := context.WithDeadline(baseCtx, imagePullDeadline)

	imageOutFile, err := os.Open(fmt.Sprintf("/clabennetes/.image/%s.tar", imageName))
	if err != nil {

	}

	err = client.Export(exportCtx, imageOutFile)
	exportCtxCancel()

	if err != nil {

	}
}

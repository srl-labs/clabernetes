package launcher

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func extractContainerlabBin(r io.Reader) error {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	defer func() {
		_ = gzipReader.Close()
	}()

	tarReader := tar.NewReader(gzipReader)

	f, err := os.OpenFile(
		"/usr/bin/containerlab",
		os.O_CREATE|os.O_RDWR,
		clabernetesconstants.PermissionsEveryoneAllPermissions,
	)
	if err != nil {
		return err
	}

	defer func() {
		_ = f.Close()
	}()

	for {
		var h *tar.Header

		h, err = tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if h.Name != "containerlab" {
			// not the clab bin, we don't care
			continue
		}

		_, err = io.Copy(f, tarReader) //nolint: gosec
		if err != nil {
			return err
		}

		return nil
	}
}

func (c *clabernetes) installContainerlabVersion(version string) error {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	tarName := fmt.Sprintf("containerlab_%s_Linux_amd64.tar.gz", version)

	outTarFile, err := os.Create(fmt.Sprintf("%s/%s", dir, tarName))
	if err != nil {
		return err
	}

	err = clabernetesutil.WriteHTTPContentsFromPath(
		context.Background(),
		fmt.Sprintf(
			"https://github.com/srl-labs/containerlab/releases/download/v%s/%s",
			version,
			tarName,
		),
		outTarFile,
		nil,
	)
	if err != nil {
		return err
	}

	inTarFile, err := os.Open(fmt.Sprintf("%s/%s", dir, tarName))
	if err != nil {
		return err
	}

	return extractContainerlabBin(inTarFile)
}

func (c *clabernetes) runContainerlab() error {
	containerlabLogFile, err := os.Create("containerlab.log")
	if err != nil {
		return err
	}

	containerlabOutWriter := io.MultiWriter(c.containerlabLogger, containerlabLogFile)

	args := []string{
		"deploy",
		"-c",
		"-t",
		"topo.clab.yaml",
	}

	if os.Getenv(clabernetesconstants.LauncherContainerlabDebug) == clabernetesconstants.True {
		args = append(args, "--debug")
	}

	containerlabTimeout := os.Getenv(clabernetesconstants.LauncherContainerlabTimeout)
	if containerlabTimeout != "" {
		args = append(args, []string{"--timeout", containerlabTimeout}...)
	}

	cmd := exec.Command("containerlab", args...)

	cmd.Stdout = containerlabOutWriter
	cmd.Stderr = containerlabOutWriter

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

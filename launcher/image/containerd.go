package image

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

func (m *containerdManager) Export(imageName, destination string) error {
	// Get the part after the last /
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
		m.logger.Infof("Export of image %q took %s", imageName, elapsed)
	}()
	// attempt to re-pull the image -- for containerd setups that have `discard_unpacked_layers`
	// set to true we will not be able to export the image as for whatever reason containerd wants
	// all layers to be present to export (even if we already ran this image on the node via the
	// puller pod!?!)

	// 1. Check if the safeName.tar already exists and lock does not exist
	if _, err := os.Stat(tarPath); err == nil {
		// .tar exists, but also check that the lock does not exist
		if _, lockErr := os.Stat(lockPath); os.IsNotExist(lockErr) {
			m.logger.Debugf("image %q was already exported!", safeName)
			return nil
		}
		// .tar exists but lock also exists, so possibly incomplete export; skip lock acquisition and proceed to wait logic below
	} else {
		// 2. Try to acquire the lock only if .tar does not exist
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// This pod is the "leader"
			lockFile.Close()
			defer os.Remove(lockPath) // clean up the lock after we're done

			// Double-check: maybe another pod already finished while we were acquiring lock
			if _, err := os.Stat(tarPath); err == nil {
				// .tar appeared, check for lock file
				if _, lockErr := os.Stat(lockPath); os.IsNotExist(lockErr) {
					m.logger.Debugf("image %q was already exported!", safeName)
					return nil
				}
			}
			exportCmd := exec.Command(
				"nerdctl",
				"--address",
				"/clabernetes/.node/containerd.sock",
				"--namespace",
				"k8s.io",
				"image",
				"save",
				"--output",
				tarPath,
				imageName,
			)

			exportCmd.Stdout = m.logger
			exportCmd.Stderr = m.logger

			err = exportCmd.Run()
			if err != nil {
				m.logger.Criticalf("failed to export image %q: %v", safeName, err)
				return err
			}

			m.logger.Debugf("image %q exported from containerd successfully...", safeName)
			return nil // Successfully exported, exit here
		}
	}
	// 3. .tar is missing but we didn't get the lock, wait up to 2 minutes for the .tar to appear and the lock to be released, checking every 3 seconds
	timeout := 2 * time.Minute
	checkInterval := 3 * time.Second
	deadline := time.Now().Add(timeout)
	for {
		// Check that tar exists AND lock does not exist
		if _, err := os.Stat(tarPath); err == nil {
			if _, lockErr := os.Stat(lockPath); os.IsNotExist(lockErr) {
				m.logger.Debugf("image %q was already exported!", safeName)
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for exported image tarball %s to appear", tarPath)
		}
		time.Sleep(checkInterval)
	}
}

func (m *containerdManager) pull(imageName string) error {
	pullCmd := exec.Command(
		"nerdctl",
		"--address",
		"/clabernetes/.node/containerd.sock",
		"--namespace",
		"k8s.io",
		"image",
		"pull",
		"--all-platforms",
		imageName,
	)

	pullCmd.Stdout = m.logger
	pullCmd.Stderr = m.logger

	err := pullCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

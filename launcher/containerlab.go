package launcher

import (
	"io"
	"os"
	"os/exec"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
)

func (c *clabernetes) runContainerlab() error {
	containerlabLogFile, err := os.Create("containerlab.log")
	if err != nil {
		return err
	}

	containerlabOutWriter := io.MultiWriter(c.containerlabLogger, containerlabLogFile)

	args := []string{
		"deploy",
		"-t",
		"topo.clab.yaml",
	}

	if os.Getenv(clabernetesconstants.LauncherContainerlabDebug) == clabernetesconstants.True {
		args = append(args, "--debug")
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

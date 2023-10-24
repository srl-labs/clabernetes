package launcher

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
)

const (
	resolveServiceMaxAttempts = 5
	resolveServiceSleep       = 10 * time.Second
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

func (c *clabernetes) runContainerlabVxlanTools(
	localNodeName, cntLink, vxlanRemote string,
	vxlanID int,
) error {
	var resolvedVxlanRemotes []net.IP

	var err error

	for attempt := 0; attempt < resolveServiceMaxAttempts; attempt++ {
		resolvedVxlanRemotes, err = net.LookupIP(vxlanRemote)
		if err != nil {
			if attempt < resolveServiceMaxAttempts {
				c.logger.Warnf(
					"failed resolving remote vxlan endpoint but under max attempts will try"+
						" again in %s. error: %s",
					resolveServiceSleep,
					err,
				)

				time.Sleep(resolveServiceSleep)

				continue
			}

			return err
		}

		break
	}

	if len(resolvedVxlanRemotes) != 1 {
		return fmt.Errorf(
			"%w: did not get exactly one ip resolved for remote vxlan endpoint",
			claberneteserrors.ErrConnectivity,
		)
	}

	resolvedVxlanRemote := resolvedVxlanRemotes[0].String()

	c.logger.Debugf("resolved remote vxlan tunnel service address as '%s'", resolvedVxlanRemote)

	cmd := exec.Command( //nolint:gosec
		"containerlab",
		"tools",
		"vxlan",
		"create",
		"--remote",
		resolvedVxlanRemote,
		"--id",
		strconv.Itoa(vxlanID),
		"--link",
		fmt.Sprintf("%s-%s", localNodeName, cntLink),
		"--port",
		strconv.Itoa(clabernetesconstants.VXLANServicePort),
	)

	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

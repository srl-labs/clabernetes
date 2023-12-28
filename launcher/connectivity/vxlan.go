package connectivity

import (
	"fmt"
	"net"
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

type vxlanManager struct {
	*common
}

func (m *vxlanManager) Run() {
	m.logger.Info(
		"connectivity mode is 'vxlan', setting up any required tunnels...",
	)

	for _, tunnel := range m.initialTunnels {
		err := m.runContainerlabVxlanTools(
			tunnel.LocalNode,
			tunnel.LocalInterface,
			tunnel.Destination,
			tunnel.TunnelID,
		)
		if err != nil {
			m.logger.Fatalf(
				"failed setting up tunnel to remote node '%s' for local interface '%s', error: %s",
				tunnel.RemoteNode,
				tunnel.LocalInterface,
				err,
			)
		}
	}

	m.logger.Debug("vxlan tunnel creation complete")
}

func (m *vxlanManager) runContainerlabVxlanTools(
	localNodeName, cntLink, vxlanRemote string,
	vxlanID int,
) error {
	var resolvedVxlanRemotes []net.IP

	var err error

	for attempt := 0; attempt < resolveServiceMaxAttempts; attempt++ {
		resolvedVxlanRemotes, err = net.LookupIP(vxlanRemote)
		if err != nil {
			if attempt < resolveServiceMaxAttempts {
				m.logger.Warnf(
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

	m.logger.Debugf("resolved remote vxlan tunnel service address as '%s'", resolvedVxlanRemote)

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

	m.logger.Debugf(
		"using following args for vxlan tunnel creation (via containerlab) '%s'", cmd.Args,
	)

	cmd.Stdout = m.logger
	cmd.Stderr = m.logger

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

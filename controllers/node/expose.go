package node

import (
	"fmt"
	"strings"

	clabernetesapisv1alpha1 "github.com/clabernetes/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/clabernetes/clabernetes/constants"
)

type managementPortDefinition struct {
	DestinationPort int
	Protocol        string
	AppProtocol     string
}

// defaultManagementPorts is the single source for the auto-expose allocation, management
// translation, and Service application-protocol defaults. Keep the order stable for status and
// rendered Service compatibility.
func defaultManagementPorts() []managementPortDefinition {
	return []managementPortDefinition{
		{clabernetesconstants.PortFTP, clabernetesconstants.TCP, "ftp"},
		{clabernetesconstants.PortSSH, clabernetesconstants.TCP, "ssh"},
		{clabernetesconstants.PortTelnet, clabernetesconstants.TCP, "telnet"},
		{clabernetesconstants.PortHTTP, clabernetesconstants.TCP, "http"},
		{clabernetesconstants.PortHTTPS, clabernetesconstants.TCP, "https"},
		{clabernetesconstants.PortNETCONF, clabernetesconstants.TCP, "netconf-ssh"},
		{clabernetesconstants.PortQemuTelnet, clabernetesconstants.TCP, "telnet"},
		{clabernetesconstants.PortVNC, clabernetesconstants.TCP, "rfb"},
		{clabernetesconstants.PortGNMIArista, clabernetesconstants.TCP, "c9s.run/gnmi"},
		{clabernetesconstants.PortGNMI, clabernetesconstants.TCP, "c9s.run/gnmi"},
		{clabernetesconstants.PortGRIBI, clabernetesconstants.TCP, "c9s.run/gribi"},
		{clabernetesconstants.PortP4RT, clabernetesconstants.TCP, "c9s.run/p4runtime"},
		{clabernetesconstants.PortGNMINokia, clabernetesconstants.TCP, "c9s.run/gnmi"},
		{clabernetesconstants.PortSNMP, clabernetesconstants.UDP, "snmp"},
	}
}

// defaultExposePorts returns the destination ports (and protocols) that get exposed
// automagically when auto expose is not disabled.
func defaultExposePorts() []clabernetesapisv1alpha1.NodeExposedPort {
	definitions := defaultManagementPorts()
	ports := make([]clabernetesapisv1alpha1.NodeExposedPort, 0, len(definitions))

	for _, definition := range definitions {
		ports = append(ports, clabernetesapisv1alpha1.NodeExposedPort{
			DestinationPort: definition.DestinationPort,
			Protocol:        definition.Protocol,
		})
	}

	return ports
}

func defaultAppProtocol(destinationPort int, protocol string) string {
	for _, definition := range defaultManagementPorts() {
		if definition.DestinationPort == destinationPort &&
			definition.Protocol == strings.ToUpper(protocol) {
			return definition.AppProtocol
		}
	}

	return ""
}

func resolvedAppProtocol(
	node *clabernetesapisv1alpha1.Node,
	port clabernetesapisv1alpha1.NodeExposedPort,
) *string {
	appProtocol := defaultAppProtocol(port.DestinationPort, port.Protocol)
	portKey := fmt.Sprintf("%d/%s", port.DestinationPort, strings.ToLower(port.Protocol))

	for _, override := range node.Spec.AppProtocols {
		if override.Port == portKey {
			appProtocol = override.AppProtocol

			break
		}
	}

	if appProtocol == "" {
		return nil
	}

	return new(appProtocol)
}

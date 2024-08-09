package topology

import (
	"fmt"
	"strings"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
)

type containerlabDefinitionProcessor struct {
	*definitionProcessor
}

func (p *containerlabDefinitionProcessor) Process() error {
	// load the containerlab topo from the CR to make sure its all good
	containerlabConfig, err := clabernetesutilcontainerlab.LoadContainerlabConfig(
		p.topology.Spec.Definition.Containerlab,
	)
	if err != nil {
		p.logger.Criticalf("failed parsing containerlab config, error: %s", err)

		return err
	}

	// we may have *different defaults per "sub-topology" so we do a cheater "deep copy" by just
	// marshalling here and unmarshalling per node in the process func :)
	defaultsYAML, err := yaml.Marshal(containerlabConfig.Topology.Defaults)
	if err != nil {
		return err
	}

	// check this here so we only have to check it once
	removeTopologyPrefix := p.getRemoveTopologyPrefix()

	for nodeName := range containerlabConfig.Topology.Nodes {
		err = p.processConfigForNode(
			containerlabConfig,
			nodeName,
			defaultsYAML,
			removeTopologyPrefix,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func getDefaultPorts() []*clabernetesutilcontainerlab.TypedPort {
	return []*clabernetesutilcontainerlab.TypedPort{
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortFTP,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortSSH,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortTelnet,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortHTTP,
		},
		{
			Protocol:        clabernetesconstants.UDP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortSNMP,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortHTTPS,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortNETCONF,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortQemuTelnet,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortVNC,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortGNMIArista,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortGNMI,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortGRIBI,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortP4RT,
		},
		{
			Protocol:        clabernetesconstants.TCP,
			ExposePort:      0,
			DestinationPort: clabernetesconstants.PortGNMINokia,
		},
	}
}

func getNextPort(allocatedPorts []int64) int64 {
	for possiblePort := 60_000; possiblePort < 65_000; possiblePort++ {
		var possiblePortFound bool

		for _, allocatedPort := range allocatedPorts {
			if int64(possiblePort) == allocatedPort {
				possiblePortFound = true

				break
			}
		}

		if !possiblePortFound {
			return int64(possiblePort)
		}
	}

	return 0
}

func typedPortsFromPortDefinitions(
	portDefinitions []string,
) []*clabernetesutilcontainerlab.TypedPort {
	typedPorts := make([]*clabernetesutilcontainerlab.TypedPort, len(portDefinitions))

	for idx, portDefinition := range portDefinitions {
		typedPort, _ := clabernetesutilcontainerlab.ProcessPortDefinition(portDefinition)

		typedPorts[idx] = typedPort
	}

	return typedPorts
}

func recordAllocatedPorts(
	typedPortDefinitions ...[]*clabernetesutilcontainerlab.TypedPort,
) (allocatedTCPExposePorts, allocatedUDPExposePorts []int64) {
	allocatedTCPExposePorts = make([]int64, 0)
	allocatedUDPExposePorts = make([]int64, 0)

	for _, typedPortDefinitionSlice := range typedPortDefinitions {
		for _, typedPort := range typedPortDefinitionSlice {
			if typedPort.ExposePort == 0 {
				continue
			}

			switch typedPort.Protocol {
			case clabernetesconstants.TCP:
				allocatedTCPExposePorts = append(allocatedTCPExposePorts, typedPort.ExposePort)
			case clabernetesconstants.UDP:
				allocatedUDPExposePorts = append(allocatedUDPExposePorts, typedPort.ExposePort)
			}
		}
	}

	return allocatedTCPExposePorts, allocatedUDPExposePorts
}

func insertMissingDefaultPorts(
	typedDefaultPorts, typedNodePorts []*clabernetesutilcontainerlab.TypedPort,
) []*clabernetesutilcontainerlab.TypedPort {
	for _, defaultTypedPort := range getDefaultPorts() {
		var alreadyDefined bool

		for _, typedPort := range typedDefaultPorts {
			if typedPort.DestinationPort != defaultTypedPort.DestinationPort {
				continue
			}

			if typedPort.Protocol != defaultTypedPort.Protocol {
				continue
			}

			alreadyDefined = true

			break
		}

		if alreadyDefined {
			continue
		}

		for _, typedPort := range typedNodePorts {
			if typedPort.DestinationPort != defaultTypedPort.DestinationPort {
				continue
			}

			if typedPort.Protocol != defaultTypedPort.Protocol {
				continue
			}

			alreadyDefined = true

			break
		}

		if alreadyDefined {
			continue
		}

		typedDefaultPorts = append(typedDefaultPorts, defaultTypedPort)
	}

	return typedDefaultPorts
}

func processPorts(
	topologyDefaultPorts, topologyNodePorts []string,
) (defaultPortsAsString, nodePortsAsString []string) {
	typedDefaultPorts := typedPortsFromPortDefinitions(topologyDefaultPorts)
	typedNodePorts := typedPortsFromPortDefinitions(topologyNodePorts)

	allocatedTCPExposePorts, allocatedUDPExposePorts := recordAllocatedPorts(
		typedDefaultPorts,
		typedNodePorts,
	)

	typedDefaultPorts = insertMissingDefaultPorts(typedDefaultPorts, typedNodePorts)

	defaultPortsAsString = make([]string, len(typedDefaultPorts))
	nodePortsAsString = make([]string, len(typedNodePorts))

	// now we go through and allocated "expose" ports for any ports that don't have this already
	// set -- unlike containerlab by default we need to know the expose ports ahead of time to
	// properly set up the lb bits.
	for idx, typedPort := range typedDefaultPorts {
		if typedPort.ExposePort != 0 {
			defaultPortsAsString[idx] = typedPort.AsContainerlabPortDefinition()

			continue
		}

		switch typedPort.Protocol {
		case clabernetesconstants.TCP:
			allocatedPort := getNextPort(allocatedTCPExposePorts)

			allocatedTCPExposePorts = append(allocatedTCPExposePorts, allocatedPort)

			typedPort.ExposePort = allocatedPort
		case clabernetesconstants.UDP:
			allocatedPort := getNextPort(allocatedUDPExposePorts)

			allocatedUDPExposePorts = append(allocatedUDPExposePorts, allocatedPort)

			typedPort.ExposePort = allocatedPort
		}

		defaultPortsAsString[idx] = typedPort.AsContainerlabPortDefinition()
	}

	for idx, typedPort := range typedNodePorts {
		if typedPort.ExposePort != 0 {
			nodePortsAsString[idx] = typedPort.AsContainerlabPortDefinition()

			continue
		}

		switch typedPort.Protocol {
		case clabernetesconstants.TCP:
			allocatedPort := getNextPort(allocatedTCPExposePorts)

			allocatedTCPExposePorts = append(allocatedTCPExposePorts, allocatedPort)

			typedPort.ExposePort = allocatedPort
		case clabernetesconstants.UDP:
			allocatedPort := getNextPort(allocatedUDPExposePorts)

			allocatedUDPExposePorts = append(allocatedUDPExposePorts, allocatedPort)

			typedPort.ExposePort = allocatedPort
		}

		nodePortsAsString[idx] = typedPort.AsContainerlabPortDefinition()
	}

	return defaultPortsAsString, nodePortsAsString
}

func getKindsForNode(
	clabTopo *clabernetesutilcontainerlab.Topology,
	nodeName string,
) map[string]*clabernetesutilcontainerlab.NodeDefinition {
	nodeKind := clabTopo.Defaults.Kind
	if clabTopo.Nodes[nodeName].Kind != "" {
		nodeKind = clabTopo.Nodes[nodeName].Kind
	}

	// we only want to snag our "sub topology" specific kind, otherwise we can just put nil
	// for the "kinds" part.
	if nodeKind != "" {
		nodeTopoKind, ok := clabTopo.Kinds[nodeKind]
		if ok {
			kindForNode := map[string]*clabernetesutilcontainerlab.NodeDefinition{
				nodeKind: nodeTopoKind,
			}

			if kindForNode[nodeKind].Ports == nil {
				// see util/containerlab/types; we dont want nil ports for now at least.
				kindForNode[nodeKind].Ports = []string{}
			}

			return kindForNode
		}
	}

	return nil
}

func (p *containerlabDefinitionProcessor) processConfigForNode(
	containerlabConfig *clabernetesutilcontainerlab.Config,
	nodeName string,
	defaultsYAML []byte,
	removeTopologyPrefix bool,
) error {
	deepCopiedDefaults := &clabernetesutilcontainerlab.NodeDefinition{}

	err := yaml.Unmarshal(defaultsYAML, deepCopiedDefaults)
	if err != nil {
		return err
	}

	nodeDefinition := containerlabConfig.Topology.Nodes[nodeName]

	if !p.topology.Spec.Expose.DisableExpose && !p.topology.Spec.Expose.DisableAutoExpose {
		// disable expose is *not* set and disable auto expose is *not* set, so we want to
		// automagically add our default expose ports to the topo. we'll simply tack this onto
		// the clab defaults ports list since that will get merged w/ any user defined ports
		defaultPorts, nodePorts := processPorts(
			containerlabConfig.Topology.Defaults.Ports,
			nodeDefinition.Ports,
		)

		deepCopiedDefaults.Ports = defaultPorts
		nodeDefinition.Ports = nodePorts
	} else {
		// zero value of slice is nil, that breaks deep equal checks later, so ensure we set to
		// a non-zero (but empty) slice
		nodeDefinition.Ports = []string{}
	}

	p.reconcileData.ResolvedConfigs[nodeName] = &clabernetesutilcontainerlab.Config{
		Name: fmt.Sprintf("clabernetes-%s", nodeName),
		Mgmt: containerlabConfig.Mgmt,
		Topology: &clabernetesutilcontainerlab.Topology{
			Defaults: deepCopiedDefaults,
			Kinds:    getKindsForNode(containerlabConfig.Topology, nodeName),
			Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
				nodeName: nodeDefinition,
			},
			Links: nil,
		},
		// we override existing topo prefix and set it to empty prefix - "" (rather than accept
		// what the user has provided *or* the default of "clab").
		// since prefixes are only useful when multiple labs are scheduled on the same node, and
		// that will never be the case with clabernetes, the prefix is unnecessary.
		Prefix: clabernetesutil.ToPointer(""),
	}

	for _, link := range containerlabConfig.Topology.Links {
		if len(link.Endpoints) != clabernetesapisv1alpha1.LinkEndpointElementCount {
			msg := fmt.Sprintf(
				"endpoint '%q' has wrong syntax, unexpected number of items", link.Endpoints,
			)

			p.logger.Critical(msg)

			return fmt.Errorf(
				"%w: %s", claberneteserrors.ErrParse, msg,
			)
		}

		endpointAParts := strings.Split(link.Endpoints[0], ":")
		endpointBParts := strings.Split(link.Endpoints[1], ":")

		if len(endpointAParts) != clabernetesapisv1alpha1.LinkEndpointElementCount ||
			len(endpointBParts) != clabernetesapisv1alpha1.LinkEndpointElementCount {
			msg := fmt.Sprintf(
				"endpoint '%q' has wrong syntax, bad endpoint:interface config", link.Endpoints,
			)

			p.logger.Critical(msg)

			return fmt.Errorf(
				"%w: %s", claberneteserrors.ErrParse, msg,
			)
		}

		endpointA := clabernetesapisv1alpha1.LinkEndpoint{
			NodeName:      endpointAParts[0],
			InterfaceName: endpointAParts[1],
		}
		endpointB := clabernetesapisv1alpha1.LinkEndpoint{
			NodeName:      endpointBParts[0],
			InterfaceName: endpointBParts[1],
		}

		if endpointA.NodeName != nodeName && endpointB.NodeName != nodeName {
			// link doesn't apply to this node, carry on
			continue
		}

		if endpointA.NodeName == nodeName && endpointB.NodeName == nodeName {
			// link loops back to ourselves, no need to do overlay things just append the link
			p.reconcileData.ResolvedConfigs[nodeName].Topology.Links = append(
				p.reconcileData.ResolvedConfigs[nodeName].Topology.Links,
				link,
			)

			continue
		}

		interestingEndpoint := endpointA
		uninterestingEndpoint := endpointB

		if endpointB.NodeName == nodeName {
			interestingEndpoint = endpointB
			uninterestingEndpoint = endpointA
		}

		p.reconcileData.ResolvedConfigs[nodeName].Topology.Links = append(
			p.reconcileData.ResolvedConfigs[nodeName].Topology.Links,
			&clabernetesutilcontainerlab.LinkDefinition{
				LinkConfig: clabernetesutilcontainerlab.LinkConfig{
					Endpoints: []string{
						fmt.Sprintf(
							"%s:%s",
							interestingEndpoint.NodeName,
							interestingEndpoint.InterfaceName,
						),
						fmt.Sprintf(
							"host:%s-%s",
							interestingEndpoint.NodeName,
							interestingEndpoint.InterfaceName,
						),
					},
				},
			},
		)

		p.reconcileData.ResolvedTunnels[nodeName] = append(
			p.reconcileData.ResolvedTunnels[nodeName],
			&clabernetesapisv1alpha1.PointToPointTunnel{
				LocalNode:  nodeName,
				RemoteNode: uninterestingEndpoint.NodeName,
				Destination: resolveConnectivityDestination(
					p.topology.Name,
					uninterestingEndpoint.NodeName,
					p.topology.Namespace,
					removeTopologyPrefix,
					p.configManagerGetter,
				),
				LocalInterface:  interestingEndpoint.InterfaceName,
				RemoteInterface: uninterestingEndpoint.InterfaceName,
			},
		)
	}

	return nil
}

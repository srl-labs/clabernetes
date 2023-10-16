package containerlab

import (
	"fmt"
	"strings"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"

	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	"gopkg.in/yaml.v3"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

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

func (c *Controller) processConfigForNode(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabTopo *clabernetesutilcontainerlab.Topology,
	nodeName string,
	nodeDefinition *clabernetesutilcontainerlab.NodeDefinition,
	defaultsYAML []byte,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	clabernetesTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) error {
	deepCopiedDefaults := &clabernetesutilcontainerlab.NodeDefinition{}

	err := yaml.Unmarshal(defaultsYAML, deepCopiedDefaults)
	if err != nil {
		return err
	}

	if !clab.Spec.DisableExpose && !clab.Spec.DisableAutoExpose {
		// disable expose is *not* set and disable auto expose is *not* set, so we want to
		// automagically add our default expose ports to the topo. we'll simply tack this onto
		// the clab defaults ports list since that will get merged w/ any user defined ports
		defaultPorts, nodePorts := processPorts(clabTopo.Defaults.Ports, nodeDefinition.Ports)

		deepCopiedDefaults.Ports = defaultPorts
		nodeDefinition.Ports = nodePorts
	}

	clabernetesConfigs[nodeName] = &clabernetesutilcontainerlab.Config{
		Name: fmt.Sprintf("clabernetes-%s", nodeName),
		Topology: &clabernetesutilcontainerlab.Topology{
			Defaults: deepCopiedDefaults,
			Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
				nodeName: nodeDefinition,
			},
			Links: nil,
		},
		// we override existing topo prefix and set it to empty prefix - "" (rather than accept
		// what the user has provided *or* the default of "clab").
		// since prefixes are only useful when multiple labs are scheduled on the same node, and
		// that will never be the case with clabernetes, the prefix is unnecessary.
		Prefix: clabernetesutil.StringToPointer(""),
	}

	for _, link := range clabTopo.Links {
		if len(link.Endpoints) != clabernetesapistopologyv1alpha1.LinkEndpointElementCount {
			msg := fmt.Sprintf(
				"endpoint '%q' has wrong syntax, unexpected number of items", link.Endpoints,
			)

			c.BaseController.Log.Critical(msg)

			return fmt.Errorf(
				"%w: %s", claberneteserrors.ErrParse, msg,
			)
		}

		endpointAParts := strings.Split(link.Endpoints[0], ":")
		endpointBParts := strings.Split(link.Endpoints[1], ":")

		if len(endpointAParts) != clabernetesapistopologyv1alpha1.LinkEndpointElementCount ||
			len(endpointBParts) != clabernetesapistopologyv1alpha1.LinkEndpointElementCount {
			msg := fmt.Sprintf(
				"endpoint '%q' has wrong syntax, bad endpoint:interface config", link.Endpoints,
			)

			c.BaseController.Log.Critical(msg)

			return fmt.Errorf(
				"%w: %s", claberneteserrors.ErrParse, msg,
			)
		}

		endpointA := clabernetesapistopologyv1alpha1.LinkEndpoint{
			NodeName:      endpointAParts[0],
			InterfaceName: endpointAParts[1],
		}
		endpointB := clabernetesapistopologyv1alpha1.LinkEndpoint{
			NodeName:      endpointBParts[0],
			InterfaceName: endpointBParts[1],
		}

		if endpointA.NodeName != nodeName && endpointB.NodeName != nodeName {
			// link doesn't apply to this node, carry on
			continue
		}

		if endpointA.NodeName == nodeName && endpointB.NodeName == nodeName {
			// link loops back to ourselves, no need to do overlay things just append the link
			clabernetesConfigs[nodeName].Topology.Links = append(
				clabernetesConfigs[nodeName].Topology.Links,
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

		clabernetesConfigs[nodeName].Topology.Links = append(
			clabernetesConfigs[nodeName].Topology.Links,
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

		clabernetesTunnels[nodeName] = append(
			clabernetesTunnels[nodeName],
			&clabernetesapistopologyv1alpha1.Tunnel{
				LocalNodeName:  nodeName,
				RemoteNodeName: uninterestingEndpoint.NodeName,
				RemoteName: fmt.Sprintf(
					"%s-%s-vx.%s.%s",
					clab.Name,
					uninterestingEndpoint.NodeName,
					clab.Namespace,
					clabernetescontrollerstopology.GetServiceDNSSuffix(),
				),
				LocalLinkName:  interestingEndpoint.InterfaceName,
				RemoteLinkName: uninterestingEndpoint.InterfaceName,
			},
		)
	}

	return nil
}

func (c *Controller) processConfig(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabTopo *clabernetesutilcontainerlab.Topology,
) (
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	clabernetesTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
	shouldUpdate bool,
	err error,
) {
	clabernetesConfigs = make(map[string]*clabernetesutilcontainerlab.Config)

	clabernetesTunnels = make(map[string][]*clabernetesapistopologyv1alpha1.Tunnel)

	// we may have *different defaults per "sub-topology" so we do a cheater "deep copy" by just
	// marshalling/unmarshalling :)
	defaultsYAML, err := yaml.Marshal(clabTopo.Defaults)
	if err != nil {
		return clabernetesConfigs, clabernetesTunnels, false, err
	}

	for nodeName, nodeDefinition := range clabTopo.Nodes {
		err = c.processConfigForNode(
			clab,
			clabTopo,
			nodeName,
			nodeDefinition,
			defaultsYAML,
			clabernetesConfigs,
			clabernetesTunnels,
		)
		if err != nil {
			return nil, nil, false, err
		}
	}

	clabernetesConfigsBytes, err := yaml.Marshal(clabernetesConfigs)
	if err != nil {
		return nil, nil, false, err
	}

	tunnelsBytes, err := yaml.Marshal(clabernetesTunnels)
	if err != nil {
		return nil, nil, false, err
	}

	newConfigsHash := clabernetesutil.HashBytes(clabernetesConfigsBytes)

	newTunnelsHash := clabernetesutil.HashBytes(tunnelsBytes)

	if clab.Status.ConfigsHash == newConfigsHash && clab.Status.TunnelsHash == newTunnelsHash {
		// the configs hashes match, nothing to do, should reconcile is false, and no error
		return clabernetesConfigs, clabernetesTunnels, false, nil
	}

	// if we got here we know we need to re-reconcile as the hash has changed, set the config and
	// config hash, and then return "true" (yes we should reconcile/update the object). before we
	// can do that though, we need to handle setting tunnel ids. so first we go over and re-use
	// all the existing tunnel ids by assigning matching node/interface pairs from the previous
	// status to the new tunnels... when doing so we record the allocated ids...
	clabernetescontrollerstopology.AllocateTunnelIDs(
		clab.Status.TopologyStatus.Tunnels,
		clabernetesTunnels,
	)

	clab.Status.Configs = string(clabernetesConfigsBytes)
	clab.Status.ConfigsHash = newConfigsHash
	clab.Status.Tunnels = clabernetesTunnels
	clab.Status.TunnelsHash = newTunnelsHash

	return clabernetesConfigs, clabernetesTunnels, true, nil
}

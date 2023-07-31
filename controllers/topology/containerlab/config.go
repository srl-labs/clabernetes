package containerlab

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"

	containerlabclab "github.com/srl-labs/containerlab/clab"
	containerlabtypes "github.com/srl-labs/containerlab/types"
	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
)

func (c *Controller) processConfig(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabTopo *containerlabtypes.Topology,
) (
	clabernetesConfigs map[string]*containerlabclab.Config,
	clabernetesTunnels map[string][]*clabernetesapistopology.Tunnel,
	shouldUpdate bool,
	err error,
) {
	clabernetesConfigs = make(map[string]*containerlabclab.Config)

	tunnels := make(map[string][]*clabernetesapistopology.Tunnel)

	for nodeName, nodeDefinition := range clabTopo.Nodes {
		clabernetesConfigs[nodeName] = &containerlabclab.Config{
			Name: fmt.Sprintf("clabernetes-%s", nodeName),
			Topology: &containerlabtypes.Topology{
				Nodes: map[string]*containerlabtypes.NodeDefinition{
					nodeName: nodeDefinition,
				},
				Links: nil,
			},
		}

		for _, link := range clabTopo.Links {
			if len(link.Endpoints) != clabernetesapistopology.LinkEndpointElementCount {
				msg := fmt.Sprintf(
					"endpoint '%q' has wrong syntax, unexpected number of items", link.Endpoints,
				)

				c.BaseController.Log.Critical(msg)

				return nil, nil, false, fmt.Errorf(
					"%w: %s", claberneteserrors.ErrParse, msg,
				)
			}

			endpointAParts := strings.Split(link.Endpoints[0], ":")
			endpointBParts := strings.Split(link.Endpoints[1], ":")

			if len(endpointAParts) != clabernetesapistopology.LinkEndpointElementCount ||
				len(endpointBParts) != clabernetesapistopology.LinkEndpointElementCount {
				msg := fmt.Sprintf(
					"endpoint '%q' has wrong syntax, bad endpoint:interface config", link.Endpoints,
				)

				c.BaseController.Log.Critical(msg)

				return nil, nil, false, fmt.Errorf(
					"%w: %s", claberneteserrors.ErrParse, msg,
				)
			}

			endpointA := clabernetesapistopology.LinkEndpoint{
				NodeName:      endpointAParts[0],
				InterfaceName: endpointAParts[1],
			}
			endpointB := clabernetesapistopology.LinkEndpoint{
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
				&containerlabtypes.LinkDefinition{
					LinkConfig: containerlabtypes.LinkConfig{
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

			tunnels[nodeName] = append(
				tunnels[nodeName],
				&clabernetesapistopology.Tunnel{
					LocalNodeName:  nodeName,
					RemoteNodeName: uninterestingEndpoint.NodeName,
					RemoteName: fmt.Sprintf(
						"%s-%s.%s.svc.cluster.local",
						clab.Name,
						uninterestingEndpoint.NodeName,
						clab.Namespace,
					),
					LocalLinkName:  interestingEndpoint.InterfaceName,
					RemoteLinkName: uninterestingEndpoint.InterfaceName,
				},
			)
		}
	}

	clabernetesConfigsBytes, err := yaml.Marshal(clabernetesConfigs)
	if err != nil {
		return nil, nil, false, err
	}

	newConfigsHash := clabernetesutil.HashBytes(clabernetesConfigsBytes)

	if clab.Status.ConfigsHash == newConfigsHash {
		// the configs hash matches, nothing to do, should reconcile is false, and no error
		return clabernetesConfigs, tunnels, false, nil
	}

	// if we got here we know we need to re-reconcile as the hash has changed, set the config and
	// config hash, and then return "true" (yes we should reconcile/update the object). before we
	// can do that though, we need to handle setting tunnel ids. so first we go over and re-use
	// all the existing tunnel ids by assigning matching node/interface pairs from the previous
	// status to the new tunnels... when doing so we record the allocated ids...
	allocateTunnelIDs(clab, tunnels)

	clab.Status.Configs = string(clabernetesConfigsBytes)
	clab.Status.ConfigsHash = newConfigsHash
	clab.Status.Tunnels = tunnels

	return clabernetesConfigs, tunnels, true, nil
}

func allocateTunnelIDs( //nolint:gocognit,gocyclo
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	tunnels map[string][]*clabernetesapistopology.Tunnel,
) {
	// iterate over stored (in status) tunnels and allocate previously assigned ids to any relevant
	// tunnels -- while doing so, make a map of all allocated tunnel ids so we can make sure to not
	// re-use things.
	allocatedTunnelIds := make(map[int]bool)

	for nodeName, nodeTunnels := range tunnels {
		existingNodeTunnels, ok := clab.Status.Tunnels[nodeName]
		if !ok {
			continue
		}

		for _, newTunnel := range nodeTunnels {
			for _, existingTunnel := range existingNodeTunnels {
				if newTunnel.LocalLinkName == existingTunnel.LocalLinkName &&
					newTunnel.RemoteName == existingTunnel.RemoteName {
					newTunnel.ID = existingTunnel.ID

					allocatedTunnelIds[newTunnel.ID] = true

					break
				}
			}
		}
	}

	for _, localNodeTunnels := range tunnels {
		for _, localTunnel := range localNodeTunnels {
			if localTunnel.ID != 0 {
				continue
			}

			var idToAssign int

			// iterate over the tunnels to see if this tunnels remote pair already has a vnid set,
			// if *yes* we need to re-use that vnid obviously!
			for remoteNodeName, remoteNodeTunnels := range tunnels {
				if remoteNodeName != localTunnel.RemoteNodeName {
					// this remote node name doesnt match our current tunnels remote node name
					continue
				}

				for _, remoteTunnel := range remoteNodeTunnels {
					if remoteTunnel.RemoteLinkName != localTunnel.LocalLinkName {
						// this specific tunnel does not match our local tunnel
						continue
					}

					if remoteTunnel.ID == 0 {
						// we found our remote tunnel but vnid is not set yet, so we'll just keep
						// doing our thing
						break
					}

					idToAssign = remoteTunnel.ID
				}
			}

			if idToAssign != 0 {
				localTunnel.ID = idToAssign

				continue
			}

			// dear lord this is probably ridiculous but should be fine for now... :)
			for i := 1; i < 16_000_000; i++ {
				_, ok := allocatedTunnelIds[i]
				if ok {
					// already allocated
					continue
				}

				localTunnel.ID = i
				allocatedTunnelIds[i] = true

				break
			}
		}
	}
}

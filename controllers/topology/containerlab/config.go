package containerlab

import (
	"fmt"
	"strings"

	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	"gopkg.in/yaml.v3"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func (c *Controller) processConfig(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabTopo *clabernetesutilcontainerlab.Topology,
) (
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	shouldUpdate bool,
	err error,
) {
	clabernetesConfigs = make(map[string]*clabernetesutilcontainerlab.Config)

	tunnels := make(map[string][]*clabernetesapistopologyv1alpha1.Tunnel)

	for nodeName, nodeDefinition := range clabTopo.Nodes {
		clabernetesConfigs[nodeName] = &clabernetesutilcontainerlab.Config{
			Name: fmt.Sprintf("clabernetes-%s", nodeName),
			Topology: &clabernetesutilcontainerlab.Topology{
				Defaults: clabTopo.Defaults,
				Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
					nodeName: nodeDefinition,
				},
				Links: nil,
			},
		}

		for _, link := range clabTopo.Links {
			if len(link.Endpoints) != clabernetesapistopologyv1alpha1.LinkEndpointElementCount {
				msg := fmt.Sprintf(
					"endpoint '%q' has wrong syntax, unexpected number of items", link.Endpoints,
				)

				c.BaseController.Log.Critical(msg)

				return nil, false, fmt.Errorf(
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

				return nil, false, fmt.Errorf(
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

			tunnels[nodeName] = append(
				tunnels[nodeName],
				&clabernetesapistopologyv1alpha1.Tunnel{
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
		return nil, false, err
	}

	newConfigsHash := clabernetesutil.HashBytes(clabernetesConfigsBytes)

	if clab.Status.ConfigsHash == newConfigsHash {
		// the configs hash matches, nothing to do, should reconcile is false, and no error
		return clabernetesConfigs, false, nil
	}

	// if we got here we know we need to re-reconcile as the hash has changed, set the config and
	// config hash, and then return "true" (yes we should reconcile/update the object). before we
	// can do that though, we need to handle setting tunnel ids. so first we go over and re-use
	// all the existing tunnel ids by assigning matching node/interface pairs from the previous
	// status to the new tunnels... when doing so we record the allocated ids...
	clabernetescontrollerstopology.AllocateTunnelIDs(clab.Status.TopologyStatus.Tunnels, tunnels)

	clab.Status.Configs = string(clabernetesConfigsBytes)
	clab.Status.ConfigsHash = newConfigsHash
	clab.Status.Tunnels = tunnels

	return clabernetesConfigs, true, nil
}

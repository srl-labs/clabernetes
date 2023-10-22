package kne

import (
	"fmt"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	knetopologyproto "github.com/openconfig/kne/proto/topo"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilkne "github.com/srl-labs/clabernetes/util/kne"
	"gopkg.in/yaml.v3"
)

func (c *Controller) processConfig( //nolint:funlen
	kne *clabernetesapistopologyv1alpha1.Kne,
	kneTopo *knetopologyproto.Topology,
) (
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	clabernetesTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
	shouldUpdate bool,
	err error,
) {
	clabernetesConfigs = make(map[string]*clabernetesutilcontainerlab.Config)

	tunnels := make(map[string][]*clabernetesapistopologyv1alpha1.Tunnel)

	// making many assumptions that things that are pointers are not going to be nil... since
	// basically everything in the kne topology obj is pointers
	for _, nodeDefinition := range kneTopo.Nodes {
		nodeName := nodeDefinition.Name
		kneVendor := nodeDefinition.Vendor.String()
		kneModel := nodeDefinition.Model

		containerlabKind := clabernetesutilkne.VendorModelToClabKind(kneVendor, kneModel)
		if containerlabKind == "" {
			msg := fmt.Sprintf(
				"cannot map kne vendor/model '%s/%s' for node '%s' to containerlab kind",
				kneVendor,
				kneModel,
				nodeName,
			)

			c.BaseController.Log.Critical(msg)

			return nil, nil, false, fmt.Errorf(
				"%w: %s", claberneteserrors.ErrParse, msg,
			)
		}

		image := nodeDefinition.Config.Image
		if image == "" {
			image = clabernetesutilkne.VendorModelToImage(kneVendor, kneModel)

			if image == "" {
				// still have no idea what image to use... bail out since we cant really do much
				// without that info
				msg := fmt.Sprintf(
					"cannot determine image to use for node '%s'",
					nodeName,
				)

				c.BaseController.Log.Critical(msg)

				return nil, nil, false, fmt.Errorf(
					"%w: %s", claberneteserrors.ErrParse, msg,
				)
			}
		}

		clabernetesConfigs[nodeName] = &clabernetesutilcontainerlab.Config{
			Name: fmt.Sprintf("clabernetes-%s", nodeName),
			Topology: &clabernetesutilcontainerlab.Topology{
				Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
					nodeName: {
						Kind:  containerlabKind,
						Image: image,
						Ports: clabernetesutilkne.VendorModelToDefaultPorts(kneVendor, kneModel),
					},
				},
				Links: nil,
			},
			Prefix: clabernetesutil.ToPointer(""),
		}

		if kneModel != "" {
			clabernetesConfigs[nodeName].Topology.Nodes[nodeName].Type = kneModel
		}

		for _, link := range kneTopo.Links {
			endpointA := clabernetesapistopologyv1alpha1.LinkEndpoint{
				NodeName:      link.ANode,
				InterfaceName: link.AInt,
			}
			endpointB := clabernetesapistopologyv1alpha1.LinkEndpoint{
				NodeName:      link.ZNode,
				InterfaceName: link.ZInt,
			}

			if endpointA.NodeName != nodeName && endpointB.NodeName != nodeName {
				// link doesn't apply to this node, carry on
				continue
			}

			if endpointA.NodeName == nodeName && endpointB.NodeName == nodeName {
				// link loops back to ourselves, no need to do overlay things just create the normal
				// clab link setup here
				clabernetesConfigs[nodeName].Topology.Links = append(
					clabernetesConfigs[nodeName].Topology.Links,
					&clabernetesutilcontainerlab.LinkDefinition{
						LinkConfig: clabernetesutilcontainerlab.LinkConfig{
							Endpoints: []string{
								fmt.Sprintf("%s:%s", endpointA.NodeName, endpointA.InterfaceName),
								fmt.Sprintf("%s:%s", endpointB.NodeName, endpointB.InterfaceName),
							},
						},
					},
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
						"%s-%s-vx.%s.%s",
						kne.Name,
						uninterestingEndpoint.NodeName,
						kne.Namespace,
						c.BaseController.GetServiceDNSSuffix(),
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

	if kne.Status.ConfigsHash == newConfigsHash {
		// the configs hash matches, nothing to do, should reconcile is false, and no error
		return clabernetesConfigs, tunnels, false, nil
	}

	// if we got here we know we need to re-reconcile as the hash has changed, set the config and
	// config hash, and then return "true" (yes we should reconcile/update the object). before we
	// can do that though, we need to handle setting tunnel ids. so first we go over and re-use
	// all the existing tunnel ids by assigning matching node/interface pairs from the previous
	// status to the new tunnels... when doing so we record the allocated ids...
	clabernetescontrollerstopology.AllocateTunnelIDs(kne.Status.TopologyStatus.Tunnels, tunnels)

	kne.Status.Configs = string(clabernetesConfigsBytes)
	kne.Status.ConfigsHash = newConfigsHash
	kne.Status.Tunnels = tunnels

	return clabernetesConfigs, tunnels, true, nil
}

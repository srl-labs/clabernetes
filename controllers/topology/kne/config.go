package kne

import (
	"encoding/json"
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

func mustFindNode(kneTopo *knetopologyproto.Topology, nodeName string) *knetopologyproto.Node {
	for idx := range kneTopo.Nodes {
		if kneTopo.Nodes[idx].Name != nodeName {
			continue
		}

		return kneTopo.Nodes[idx]
	}

	panic(fmt.Sprintf("could not find node definition for node '%s'", nodeName))
}

// janky helper to yoink the config file name out of a kne topo, its hidden behind some weird pb
// nonsense and the actual fields are not accessible but theyre there? or im dumb (or both!),
// so just marshall/unmarshall to get what we want easily. panics if we fail here since i don't
// really think this can/should fail... in theory!
func getConfigFile(nodeConfigDefinition *knetopologyproto.Config) string {
	topoB, err := json.Marshal(nodeConfigDefinition)
	if err != nil {
		panic(fmt.Sprintf("error marshalling node config definition, error: %s", err))
	}

	type kneConfigDataFile struct {
		File string `json:"File"`
	}

	type kneConfigData struct {
		Data kneConfigDataFile `json:"ConfigData"`
	}

	topoConfigData := &kneConfigData{}

	err = json.Unmarshal(topoB, topoConfigData)
	if err != nil {
		panic(fmt.Sprintf("error unmarshalling node config definition, error: %s", err))
	}

	return topoConfigData.Data.File
}

func (c *Controller) processConfig( //nolint:funlen
	kne *clabernetesapistopologyv1alpha1.Kne,
	kneTopo *knetopologyproto.Topology,
) (
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
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

			return nil, false, fmt.Errorf(
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

				return nil, false, fmt.Errorf(
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
		}

		if kneModel != "" {
			clabernetesConfigs[nodeName].Topology.Nodes[nodeName].Type = kneModel
		}

		configFile := getConfigFile(nodeDefinition.Config)

		if configFile != "" {
			clabernetesConfigs[nodeName].Topology.Nodes[nodeName].StartupConfig = configFile
		}

		for _, link := range kneTopo.Links {
			zNodeDefinition := mustFindNode(kneTopo, link.ZNode)

			endpointA := clabernetesapistopologyv1alpha1.LinkEndpoint{
				NodeName:      link.ANode,
				InterfaceName: nodeDefinition.Interfaces[link.AInt].Name,
			}
			endpointB := clabernetesapistopologyv1alpha1.LinkEndpoint{
				NodeName:      link.ZNode,
				InterfaceName: zNodeDefinition.Interfaces[link.ZInt].Name,
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
								fmt.Sprintf(
									"%s:%s",
									endpointA.NodeName,
									endpointA.InterfaceName,
								),
								fmt.Sprintf(
									"%s:%s",
									endpointA.NodeName,
									endpointA.InterfaceName,
								),
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
						"%s-%s.%s.svc.cluster.local",
						kne.Name,
						uninterestingEndpoint.NodeName,
						kne.Namespace,
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

	if kne.Status.ConfigsHash == newConfigsHash {
		// the configs hash matches, nothing to do, should reconcile is false, and no error
		return clabernetesConfigs, false, nil
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

	return clabernetesConfigs, true, nil
}

package topology

import (
	"fmt"
	"maps"
	"sort"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExpandedTopology is the decomposed form of a Topology: the set of Node and Link custom resources
// that, together, represent the same network as the Topology's inline definition. See
// docs/design/0001-scale-node-link-crds.md.
type ExpandedTopology struct {
	// Nodes is the set of Node objects -- one per containerlab node (or per node group that shares a
	// network namespace) -- sorted by node name for determinism.
	Nodes []*clabernetesapisv1alpha1.Node
	// Links is the set of Link objects -- one per cross-pod point-to-point link -- sorted by tunnel
	// id for determinism. Links internal to a node group, and containerlab "host" links, are not
	// represented here (they live inside the relevant Node's sub-topology).
	Links []*clabernetesapisv1alpha1.Link
}

// ExpandTopology parses the given Topology's definition and expands it into the discrete set of Node
// and Link custom resources that represent it.
//
// It is a *pure* function: it performs no cluster access and does not mutate the input Topology. It
// deliberately reuses the existing definition processor -- so the per-node sub-topologies are
// byte-for-byte identical to what the launcher pods already receive today -- and the existing tunnel
// id allocator -- so both ends of a link converge on a single, stable vxlan/slurpeeth id. This is
// the Phase 0 building block of the scalable, decomposed reconcile path; it currently has no runtime
// effect and is exercised by unit tests only.
func ExpandTopology(
	logger claberneteslogging.Instance,
	topology *clabernetesapisv1alpha1.Topology,
	resolvedDefinition string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) (*ExpandedTopology, error) {
	reconcileData, err := NewReconcileData(topology)
	if err != nil {
		return nil, err
	}

	// carry any indirectly-sourced (ConfigMap/URL) definition through to the processor so the
	// expansion matches the live reconcile path; empty when the definition is inlined directly.
	reconcileData.ResolvedDefinition = resolvedDefinition

	processor, err := NewDefinitionProcessor(
		logger,
		topology,
		reconcileData,
		configManagerGetter,
	)
	if err != nil {
		return nil, err
	}

	err = processor.Process()
	if err != nil {
		return nil, err
	}

	// allocate tunnel ids exactly the way the controller does. there are no previously allocated
	// ids to preserve here (this is a pure expansion), and allocating now guarantees that the two
	// half-tunnels of any given link share the same id -- which is what lets us collapse them into a
	// single Link object below.
	AllocateTunnelIDs(
		map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{},
		reconcileData.ResolvedTunnels,
	)

	nodes, err := expandNodes(topology, reconcileData, configManagerGetter)
	if err != nil {
		return nil, err
	}

	links := expandLinks(topology, reconcileData, configManagerGetter)

	return &ExpandedTopology{
		Nodes: nodes,
		Links: links,
	}, nil
}

// expandNodes renders one Node object per resolved sub-topology (one per node group).
func expandNodes(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) ([]*clabernetesapisv1alpha1.Node, error) {
	annotations, globalLabels := configManagerGetter().GetAllMetadata()

	nodeNames := make([]string, 0, len(reconcileData.ResolvedConfigs))
	for nodeName := range reconcileData.ResolvedConfigs {
		nodeNames = append(nodeNames, nodeName)
	}

	sort.Strings(nodeNames)

	nodes := make([]*clabernetesapisv1alpha1.Node, 0, len(nodeNames))

	for _, nodeName := range nodeNames {
		definitionBytes, err := yaml.Marshal(reconcileData.ResolvedConfigs[nodeName])
		if err != nil {
			return nil, err
		}

		var filesFromConfigMap []clabernetesapisv1alpha1.FileFromConfigMap
		if topology.Spec.Deployment.FilesFromConfigMap != nil {
			filesFromConfigMap = topology.Spec.Deployment.FilesFromConfigMap[nodeName]
		}

		var filesFromURL []clabernetesapisv1alpha1.FileFromURL
		if topology.Spec.Deployment.FilesFromURL != nil {
			filesFromURL = topology.Spec.Deployment.FilesFromURL[nodeName]
		}

		nodes = append(
			nodes,
			&clabernetesapisv1alpha1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        expandedNodeName(topology.GetName(), nodeName),
					Namespace:   topology.GetNamespace(),
					Annotations: annotations,
					Labels: expandedLabels(
						globalLabels,
						topology.GetName(),
						reconcileData.Kind,
					),
				},
				Spec: clabernetesapisv1alpha1.NodeSpec{
					TopologyName:       topology.GetName(),
					NodeName:           nodeName,
					Kind:               reconcileData.Kind,
					Definition:         string(definitionBytes),
					Connectivity:       topology.Spec.Connectivity,
					FilesFromConfigMap: filesFromConfigMap,
					FilesFromURL:       filesFromURL,
				},
			},
		)
	}

	return nodes, nil
}

// expandLinks collapses the per-node half-tunnels produced by the definition processor into one
// Link object per cross-pod link. Each half-tunnel appears once under each endpoint's node; we
// canonicalize the endpoint ordering so the two halves map to the same Link, and we take the tunnel
// id from the half-tunnel (both halves share it after AllocateTunnelIDs).
func expandLinks(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) []*clabernetesapisv1alpha1.Link {
	annotations, globalLabels := configManagerGetter().GetAllMetadata()

	return buildLinks(topology, reconcileData, annotations, globalLabels)
}

// buildLinks collapses the per-node half-tunnels in reconcileData into one Link object per cross-pod
// link. It assumes tunnel ids have already been allocated on reconcileData (so both halves of a link
// share an id); callers that need stable ids across reconciles must run AllocateTunnelIDs first. It
// is shared by the pure ExpandTopology path and the orchestrator's ReconcileLinks.
func buildLinks(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
	annotations, globalLabels map[string]string,
) []*clabernetesapisv1alpha1.Link {
	primaries := make([]string, 0, len(reconcileData.ResolvedTunnels))
	for primary := range reconcileData.ResolvedTunnels {
		primaries = append(primaries, primary)
	}

	sort.Strings(primaries)

	seen := make(map[string]bool)
	links := make([]*clabernetesapisv1alpha1.Link, 0)

	for _, primary := range primaries {
		for _, tunnel := range reconcileData.ResolvedTunnels[primary] {
			endpointA := clabernetesapisv1alpha1.LinkEndpoint{
				NodeName:      tunnel.LocalNode,
				InterfaceName: tunnel.LocalInterface,
			}
			endpointB := clabernetesapisv1alpha1.LinkEndpoint{
				NodeName:      tunnel.RemoteNode,
				InterfaceName: tunnel.RemoteInterface,
			}

			// canonicalize endpoint order so the two half-tunnels of a link collapse to one Link.
			if !endpointLess(endpointA, endpointB) {
				endpointA, endpointB = endpointB, endpointA
			}

			key := fmt.Sprintf(
				"%s:%s|%s:%s",
				endpointA.NodeName, endpointA.InterfaceName,
				endpointB.NodeName, endpointB.InterfaceName,
			)
			if seen[key] {
				continue
			}

			seen[key] = true

			links = append(
				links,
				&clabernetesapisv1alpha1.Link{
					ObjectMeta: metav1.ObjectMeta{
						Name:        expandedLinkName(topology.GetName(), tunnel.TunnelID),
						Namespace:   topology.GetNamespace(),
						Annotations: annotations,
						Labels: expandedLabels(
							globalLabels,
							topology.GetName(),
							reconcileData.Kind,
						),
					},
					Spec: clabernetesapisv1alpha1.LinkSpec{
						TopologyName: topology.GetName(),
						EndpointA:    endpointA,
						EndpointB:    endpointB,
						Connectivity: topology.Spec.Connectivity,
						TunnelID:     tunnel.TunnelID,
					},
				},
			)
		}
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].Spec.TunnelID < links[j].Spec.TunnelID
	})

	return links
}

// endpointLess provides a total ordering over link endpoints so endpoint pairs can be canonicalized.
func endpointLess(a, b clabernetesapisv1alpha1.LinkEndpoint) bool {
	if a.NodeName != b.NodeName {
		return a.NodeName < b.NodeName
	}

	return a.InterfaceName < b.InterfaceName
}

// expandedNodeName returns the (deterministic) name for a Node object. NOTE: naming here is
// provisional -- the final scheme (including the prefixed/non-prefixed Topology naming modes) is
// settled in Phase 1; see docs/design/0001-scale-node-link-crds.md.
func expandedNodeName(topologyName, nodeName string) string {
	return fmt.Sprintf("%s-%s", topologyName, nodeName)
}

// expandedLinkName returns the (deterministic) name for a Link object, keyed on the unique tunnel
// id. Naming here is provisional, see expandedNodeName.
func expandedLinkName(topologyName string, tunnelID int) string {
	return fmt.Sprintf("%s-link-%d", topologyName, tunnelID)
}

// PerNodeConnectivityName returns the (deterministic) name of the per-node Connectivity object used
// by the decomposed reconcile path. Each node gets its own small Connectivity object (holding only
// that node's tunnels) so no single object grows with topology size -- replacing the one
// topology-wide Connectivity. The Node controller points the launcher at this object via
// LauncherConnectivityNameEnv; both sides derive the name from here so they always agree.
func PerNodeConnectivityName(topologyName, nodeName string) string {
	return fmt.Sprintf("%s-connectivity", expandedNodeName(topologyName, nodeName))
}

// expandedLabels builds the standard clabernetes owner labels (mirroring the existing sub-reconciler
// label set) merged with any globally configured labels.
func expandedLabels(globalLabels map[string]string, topologyName, kind string) map[string]string {
	labels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          topologyName,
		clabernetesconstants.LabelTopologyOwner: topologyName,
		clabernetesconstants.LabelTopologyKind:  kind,
	}

	maps.Copy(labels, globalLabels)

	return labels
}

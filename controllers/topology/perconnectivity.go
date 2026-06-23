package topology

import (
	"context"
	"sort"

	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilePerNodeConnectivity is the decomposed replacement for ReconcileConnectivity: instead of a
// single topology-wide Connectivity object holding every node's tunnels (which grows with topology
// size and eventually blows the etcd object ceiling), it writes one small Connectivity object per
// node, holding only that node's tunnels. Each node's launcher is pointed at its own object by the
// Node controller. It also reconciles the Link ledger and prunes any stale (or legacy monolithic)
// Connectivity objects. Gated behind spec.deployment.decompose; see
// docs/design/0001-scale-node-link-crds.md.
func (r *Reconciler) ReconcilePerNodeConnectivity(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	existingConnectivityList := &clabernetesapisv1alpha1.ConnectivityList{}

	err := r.Client.List(
		ctx,
		existingConnectivityList,
		ctrlruntimeclient.InNamespace(owningTopology.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: owningTopology.GetName(),
		},
	)
	if err != nil {
		return err
	}

	// gather every previously allocated tunnel id (from whatever connectivity objects exist -- the
	// per-node ones and/or a legacy monolithic one) so AllocateTunnelIDs preserves them; renumbering
	// a live tunnel would tear down a working link.
	previousTunnels := map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{}

	for idx := range existingConnectivityList.Items {
		for nodeName, tunnels := range existingConnectivityList.Items[idx].Spec.PointToPointTunnels {
			previousTunnels[nodeName] = tunnels
		}
	}

	AllocateTunnelIDs(previousTunnels, reconcileData.ResolvedTunnels)

	// reconcile the Link ledger now that ids are stable.
	err = r.ReconcileLinks(ctx, owningTopology, reconcileData)
	if err != nil {
		return err
	}

	// one Connectivity object per node -- iterate over all resolved nodes (not just those with
	// tunnels) so every launcher has an object to read, even if its tunnel list is empty.
	nodeNames := make([]string, 0, len(reconcileData.ResolvedConfigs))
	for nodeName := range reconcileData.ResolvedConfigs {
		nodeNames = append(nodeNames, nodeName)
	}

	sort.Strings(nodeNames)

	desired := make(map[string]bool, len(nodeNames))

	for _, nodeName := range nodeNames {
		name := PerNodeConnectivityName(owningTopology.GetName(), nodeName)
		desired[name] = true

		rendered := r.connectivityReconciler.Render(
			owningTopology,
			map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{
				nodeName: reconcileData.ResolvedTunnels[nodeName],
			},
		)
		rendered.Name = name

		err = r.createOrUpdatePerNodeConnectivity(ctx, owningTopology, rendered)
		if err != nil {
			return err
		}
	}

	// prune any connectivity objects owned by this topology that we no longer want -- stale per-node
	// objects and the legacy topology-wide object (named after the topology) alike.
	for idx := range existingConnectivityList.Items {
		existing := &existingConnectivityList.Items[idx]
		if desired[existing.GetName()] {
			continue
		}

		err = r.deleteObj(ctx, existing, clabernetesapis.Connectivity)
		if err != nil {
			return err
		}
	}

	return nil
}

// createOrUpdatePerNodeConnectivity creates or updates a single rendered per-node Connectivity
// object, reusing the shared Connectivity conformance check.
func (r *Reconciler) createOrUpdatePerNodeConnectivity(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	rendered *clabernetesapisv1alpha1.Connectivity,
) error {
	existing := &clabernetesapisv1alpha1.Connectivity{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: rendered.GetNamespace(),
			Name:      rendered.GetName(),
		},
		existing,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.createObj(ctx, owningTopology, rendered, clabernetesapis.Connectivity)
		}

		return err
	}

	if r.connectivityReconciler.Conforms(existing, rendered, owningTopology.GetUID()) {
		return nil
	}

	rendered.ResourceVersion = existing.ResourceVersion

	return r.updateObj(ctx, rendered, clabernetesapis.Connectivity)
}

package topology

import (
	"context"
	"reflect"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// nodeKind is the (logging) kind string used when creating/updating/deleting Node objects.
const nodeKind = "node"

// ReconcileNodes is the orchestrator half of the decomposed (scalable) reconcile path: it expands
// the owning Topology into the desired set of Node objects and reconciles (creates/updates/prunes)
// them. Each Node is then reconciled independently by the Node controller. This is only invoked when
// the Topology opts in via spec.deployment.decompose; see docs/design/0001-scale-node-link-crds.md.
func (r *Reconciler) ReconcileNodes(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	expanded, err := ExpandTopology(
		r.Log,
		owningTopology,
		reconcileData.ResolvedDefinition,
		clabernetesconfig.GetManager,
	)
	if err != nil {
		return err
	}

	existingNodeList := &clabernetesapisv1alpha1.NodeList{}

	err = r.Client.List(
		ctx,
		existingNodeList,
		ctrlruntimeclient.InNamespace(owningTopology.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: owningTopology.GetName(),
		},
	)
	if err != nil {
		return err
	}

	desired := make(map[string]*clabernetesapisv1alpha1.Node, len(expanded.Nodes))
	for _, node := range expanded.Nodes {
		desired[node.GetName()] = node
	}

	existing := make(map[string]*clabernetesapisv1alpha1.Node, len(existingNodeList.Items))
	for idx := range existingNodeList.Items {
		existing[existingNodeList.Items[idx].GetName()] = &existingNodeList.Items[idx]
	}

	// prune Node objects that no longer exist in the topology.
	for name, node := range existing {
		if _, ok := desired[name]; ok {
			continue
		}

		err = r.deleteObj(ctx, node, nodeKind)
		if err != nil {
			return err
		}
	}

	// create missing / update changed Node objects.
	for name, desiredNode := range desired {
		currentNode, ok := existing[name]
		if !ok {
			err = r.createObj(ctx, owningTopology, desiredNode, nodeKind)
			if err != nil {
				return err
			}

			continue
		}

		if reflect.DeepEqual(currentNode.Spec, desiredNode.Spec) &&
			mapContainsAll(currentNode.Labels, desiredNode.Labels) {
			continue
		}

		currentNode.Spec = desiredNode.Spec

		if currentNode.Labels == nil {
			currentNode.Labels = map[string]string{}
		}

		for k, v := range desiredNode.Labels {
			currentNode.Labels[k] = v
		}

		err = r.updateObj(ctx, currentNode, nodeKind)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileNodeStatuses is the decomposed-path equivalent of the readiness rollup that the monolithic
// path does inside ReconcileDeployments: it reads the owned Node objects' reported readiness and
// aggregates it up to the Topology status (NodeReadiness, the TopologyReady condition, TopologyState).
// In the decomposed path the Topology no longer manages the deployments directly -- the Node
// controller does -- so the Topology learns each node's readiness from the Node's own status. Gated;
// see docs/design/0001-scale-node-link-crds.md.
func (r *Reconciler) ReconcileNodeStatuses(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	existingNodeList := &clabernetesapisv1alpha1.NodeList{}

	err := r.Client.List(
		ctx,
		existingNodeList,
		ctrlruntimeclient.InNamespace(owningTopology.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: owningTopology.GetName(),
		},
	)
	if err != nil {
		return err
	}

	for idx := range existingNodeList.Items {
		node := &existingNodeList.Items[idx]

		state := node.Status.Readiness
		if state == "" {
			// node exists but hasn't been reconciled yet -- treat as unknown for now.
			state = clabernetesconstants.NodeStatusUnknown
		}

		reconcileData.NodeStatuses[node.Spec.NodeName] = state
	}

	r.applyTopologyReadiness(owningTopology, reconcileData)

	return nil
}

// mapContainsAll returns true if have contains every key/value pair in want.
func mapContainsAll(have, want map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}

	return true
}

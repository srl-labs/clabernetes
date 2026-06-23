package topology

import (
	"context"
	"reflect"

	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileLinks is the orchestrator half of per-link decomposition: it turns the resolved tunnels
// into one Link object per cross-pod link and reconciles (creates/updates/prunes) them. The Link
// objects are the durable, distributed ledger of tunnel-id allocations -- each Link carries its own
// spec.tunnelID, so no single object grows with topology size. This is only invoked when the
// Topology opts in via spec.deployment.decompose; see docs/design/0001-scale-node-link-crds.md.
//
// NOTE: callers must have already run AllocateTunnelIDs on reconcileData (ReconcilePerNodeConnectivity
// does this) so the ids baked into the Links match what the launchers actually use.
func (r *Reconciler) ReconcileLinks(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	annotations, globalLabels := clabernetesconfig.GetManager().GetAllMetadata()

	desiredLinks := buildLinks(owningTopology, reconcileData, annotations, globalLabels)

	existingLinkList := &clabernetesapisv1alpha1.LinkList{}

	err := r.Client.List(
		ctx,
		existingLinkList,
		ctrlruntimeclient.InNamespace(owningTopology.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: owningTopology.GetName(),
		},
	)
	if err != nil {
		return err
	}

	desired := make(map[string]*clabernetesapisv1alpha1.Link, len(desiredLinks))
	for _, link := range desiredLinks {
		desired[link.GetName()] = link
	}

	existing := make(map[string]*clabernetesapisv1alpha1.Link, len(existingLinkList.Items))
	for idx := range existingLinkList.Items {
		existing[existingLinkList.Items[idx].GetName()] = &existingLinkList.Items[idx]
	}

	// prune Link objects that no longer exist in the topology.
	for name, link := range existing {
		if _, ok := desired[name]; ok {
			continue
		}

		err = r.deleteObj(ctx, link, clabernetesapis.Link)
		if err != nil {
			return err
		}
	}

	// create missing / update changed Link objects.
	for name, desiredLink := range desired {
		currentLink, ok := existing[name]
		if !ok {
			err = r.createObj(ctx, owningTopology, desiredLink, clabernetesapis.Link)
			if err != nil {
				return err
			}

			continue
		}

		if reflect.DeepEqual(currentLink.Spec, desiredLink.Spec) &&
			mapContainsAll(currentLink.Labels, desiredLink.Labels) {
			continue
		}

		currentLink.Spec = desiredLink.Spec

		if currentLink.Labels == nil {
			currentLink.Labels = map[string]string{}
		}

		for k, v := range desiredLink.Labels {
			currentLink.Labels[k] = v
		}

		err = r.updateObj(ctx, currentLink, clabernetesapis.Link)
		if err != nil {
			return err
		}
	}

	return nil
}

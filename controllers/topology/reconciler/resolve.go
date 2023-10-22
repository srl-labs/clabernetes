package reconciler

import (
	"context"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"

	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileResolve[T ctrlruntimeclient.Object, TL ctrlruntimeclient.ObjectList](
	ctx context.Context,
	reconciler *Reconciler,
	ownedType T,
	ownedTypeListing TL,
	ownedTypeName string,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	resolveFunc func(
		ownedObject TL,
		currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
		owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	) (*clabernetesutilkubernetes.ObjectDiffer[T], error),
) (*clabernetesutilkubernetes.ObjectDiffer[T], error) {
	// strictly passed for typing reasons
	_ = ownedType

	err := reconciler.Client.List(
		ctx,
		ownedTypeListing,
		ctrlruntimeclient.InNamespace(owningTopology.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: owningTopology.GetName(),
		},
	)
	if err != nil {
		reconciler.Log.Criticalf("failed fetching owned deployments, error: '%s'", err)

		return nil, err
	}

	resolved, err := resolveFunc(ownedTypeListing, currentClabernetesConfigs, owningTopology)
	if err != nil {
		reconciler.Log.Criticalf("failed resolving owned deployments, error: '%s'", err)

		return nil, err
	}

	reconciler.Log.Debugf(
		"%ss are missing for the following nodes: %s",
		ownedTypeName,
		resolved.Missing,
	)

	reconciler.Log.Debugf(
		"extraneous %ss exist for following nodes: %s",
		ownedTypeName,
		resolved.Extra,
	)

	return resolved, nil
}

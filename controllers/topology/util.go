package topology

import (
	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
)

// GetTopologyKind returns the "kind" of topology this CR represents -- typically this will be
// "containerlab", but may be "kne" or perhaps others in the future as well.
func GetTopologyKind(t *clabernetesapisv1alpha1.Topology) string {
	if t.Spec.Definition.Kne != "" {
		return clabernetesapis.TopologyKindKne
	}

	// we should (eventually) prevent having an empty definition, at which point if it isn't kne
	// kind, itll be containerlab kind
	return clabernetesapis.TopologyKindContainerlab
}

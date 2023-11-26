package topology

import (
	"context"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// getTopologyFromReq fetches the reconcile target Topology from the Request.
func (c *Controller) getTopologyFromReq(
	ctx context.Context,
	req ctrlruntime.Request,
) (*clabernetesapisv1alpha1.Topology, error) {
	containerlab := &clabernetesapisv1alpha1.Topology{}

	err := c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
		containerlab,
	)

	return containerlab, err
}

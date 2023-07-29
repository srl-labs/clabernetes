package containerlab

import (
	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	"golang.org/x/net/context"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// getClababFromReq fetches the reconcile target containerlab topology from the Request.
func (c *Controller) getClababFromReq(
	ctx context.Context,
	req ctrlruntime.Request,
) (*clabernetesapistopologyv1alpha1.Containerlab, error) {
	clab := &clabernetesapistopologyv1alpha1.Containerlab{}

	err := c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
		clab,
	)

	return clab, err
}

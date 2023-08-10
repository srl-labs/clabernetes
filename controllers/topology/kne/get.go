package kne

import (
	"context"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// getKneFromReq fetches the reconcile target Kne topology from the Request.
func (c *Controller) getKneFromReq(
	ctx context.Context,
	req ctrlruntime.Request,
) (*clabernetesapistopologyv1alpha1.Kne, error) {
	kne := &clabernetesapistopologyv1alpha1.Kne{}

	err := c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
		kne,
	)

	return kne, err
}

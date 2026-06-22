package node

import (
	"context"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconcile handles reconciliation for this controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	node := &clabernetesapisv1alpha1.Node{}

	err := c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
		node,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			// was deleted, nothing to do -- owned child resources are cleaned up via owner refs
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if node.DeletionTimestamp != nil {
		// nothing for us to do, k8s will handle the cascade delete of our owned resources
		return ctrlruntime.Result{}, nil
	}

	err = c.NodeReconciler.ReconcileNode(ctx, node)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling node '%s/%s', error: %s",
			node.Namespace,
			node.Name,
			err,
		)

		return ctrlruntime.Result{}, err
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

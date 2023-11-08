package containerlab

import (
	"context"

	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconcile handles reconciliation for this controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	containerlab, err := c.getContainerlabFromReq(ctx, req)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			// was deleted, nothing to do
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if containerlab.DeletionTimestamp != nil {
		// deleting nothing to do, we have no finalizers or anything at this point
		return ctrlruntime.Result{}, nil
	}

	reconcileData, err := clabernetescontrollerstopologyreconciler.NewReconcileData(containerlab)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed processing previously stored containerlab resource, error: %s", err,
		)

		return ctrlruntime.Result{}, err
	}

	// load the containerlab topo from the CR to make sure its all good
	containerlabTopo, err := clabernetesutilcontainerlab.LoadContainerlabTopology(
		containerlab.Spec.Config,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing containerlab config, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.processConfig(containerlab, containerlabTopo, reconcileData)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing containerlab config, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		containerlab,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling clabernetes config map, error: %s",
			err,
		)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileServices(
		ctx,
		containerlab,
		reconcileData,
	)
	if err != nil {
		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		containerlab,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	if reconcileData.ShouldUpdateResource {
		// we should update because config hash or something changed, so snag the updated status
		// data out of the reconcile data, put it in the resource, and push the update
		reconcileData.SetStatus(&containerlab.Status.TopologyStatus)

		err = c.BaseController.Client.Update(ctx, containerlab)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed updating object '%s/%s' error: %s",
				containerlab.Namespace,
				containerlab.Name,
				err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

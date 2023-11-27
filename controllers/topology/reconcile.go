package topology

import (
	"context"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconcile handles reconciliation for this controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	topology, err := c.getTopologyFromReq(ctx, req)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			// was deleted, nothing to do
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if topology.DeletionTimestamp != nil {
		// deleting nothing to do, we have no finalizers or anything at this point
		return ctrlruntime.Result{}, nil
	}

	if c.BaseController.ShouldIgnoreReconcile(topology) {
		return ctrlruntime.Result{}, nil
	}

	reconcileData, err := NewReconcileData(topology)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed processing previously stored containerlab resource, error: %s", err,
		)

		return ctrlruntime.Result{}, err
	}

	err = c.processDefinition(topology, reconcileData)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing topology definition, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		topology,
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
		topology,
		reconcileData,
	)
	if err != nil {
		// error already logged
		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcilePersistentVolumeClaim(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes pvcs, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	if reconcileData.ShouldUpdateResource {
		// we should update because config hash or something changed, so snag the updated status
		// data out of the reconcile data, put it in the resource, and push the update
		reconcileData.SetStatus(&topology.Status)

		err = c.BaseController.Client.Update(ctx, topology)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed updating object '%s/%s' error: %s",
				topology.Namespace,
				topology.Name,
				err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

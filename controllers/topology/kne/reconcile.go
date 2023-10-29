package kne

import (
	"context"

	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"
	clabernetesutilkne "github.com/srl-labs/clabernetes/util/kne"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconcile handles reconciliation for this controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	kne, err := c.getKneFromReq(ctx, req)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			// was deleted, nothing to do
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if kne.DeletionTimestamp != nil {
		// deleting nothing to do, we have no finalizers or anything at this point
		return ctrlruntime.Result{}, nil
	}

	reconcileData, err := clabernetescontrollerstopologyreconciler.NewReconcileData(kne)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed processing previously stored kne resource, error: %s", err,
		)

		return ctrlruntime.Result{}, err
	}

	// load the kne topo to make sure its all good
	kneTopo, err := clabernetesutilkne.LoadKneTopology(kne.Spec.Topology)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing kne topology, error: ", err)

		return ctrlruntime.Result{}, err
	}

	err = c.processConfig(kne, kneTopo, reconcileData)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing kne config, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		kne,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling clabernetes config map, error: %s",
			err,
		)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileServiceFabric(ctx, kne, reconcileData)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes services, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileServicesExpose(
		ctx,
		kne,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling clabernetes expose services, error: %s", err,
		)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		kne,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	if reconcileData.ShouldUpdateResource {
		// we should update because config hash or something changed, so snag the updated status
		// data out of the reconcile data, put it in the resource, and push the update
		reconcileData.SetStatus(&kne.Status.TopologyStatus)

		err = c.BaseController.Client.Update(ctx, kne)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed updating object '%s/%s' error: %s",
				kne.Namespace,
				kne.Name,
				err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

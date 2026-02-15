package topology

import (
	"context"
	"time"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// destroyingStateObservabilityDelay is the time we hold the "destroying" state before removing
// the finalizer, giving external watchers a chance to observe the transition.
const destroyingStateObservabilityDelay = 5 * time.Second

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
		return c.handleDeletion(ctx, req, topology)
	}

	if c.BaseController.ShouldIgnoreReconcile(topology) {
		return ctrlruntime.Result{}, nil
	}

	// we always reconcile the "namespace" resources first -- meaning the resources that exist in
	// the namespace that are not 1:1 to a Topology -- for example: service account and role
	// binding. These resources are created for the namespace on creation of the first Topology in
	// the namespace, and are removed when the last Topology is removed from the namespace.
	err = c.TopologyReconciler.ReconcileNamespaceResources(ctx, topology)
	if err != nil {
		return ctrlruntime.Result{}, err
	}

	reconcileData, err := NewReconcileData(topology)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed processing previously stored containerlab resource, error: %s", err,
		)

		return ctrlruntime.Result{}, err
	}

	// reconcile the naming -- we *must* do this to ensure that our status field is set!
	c.TopologyReconciler.ReconcileNaming(topology, reconcileData)

	err = c.processDefinition(topology, reconcileData)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing topology definition, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.reconcileResources(ctx, topology, reconcileData)
	if err != nil {
		return ctrlruntime.Result{}, err
	}

	// Also force an update when the finalizer has not been added yet so we can piggyback it
	// on the same write as the valid status (avoids a separate write with empty status).
	if !controllerutil.ContainsFinalizer(topology, clabernetesconstants.TopologyFinalizer) {
		reconcileData.ShouldUpdateResource = true
	}

	if reconcileData.ShouldUpdateResource {
		// we should update because config hash or something changed, so snag the updated status
		// data out of the reconcile data, put it in the resource, and push the update
		err = reconcileData.SetStatus(&topology.Status)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed setting object '%s/%s' status, error: %s",
				topology.Namespace,
				topology.Name,
				err,
			)

			return ctrlruntime.Result{}, err
		}

		// Add the finalizer so the controller can observe deletion and set "destroying" state.
		// This is done here (not earlier) so it rides the same write as the valid status.
		controllerutil.AddFinalizer(topology, clabernetesconstants.TopologyFinalizer)

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

func (c *Controller) reconcileResources(
	ctx context.Context,
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	err := c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling clabernetes config map, error: %s",
			err,
		)

		return err
	}

	err = c.TopologyReconciler.ReconcileConnectivity(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed reconciling clabernetes connectivity resource, error: %s",
			err,
		)

		return err
	}

	err = c.TopologyReconciler.ReconcileServices(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		// error already logged
		return err
	}

	err = c.TopologyReconciler.ReconcilePersistentVolumeClaim(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes pvcs, error: %s", err)

		return err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		topology,
		reconcileData,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return err
	}

	return nil
}

// handleDeletion implements a two-pass deletion so that external watchers can
// observe the "destroying" state before the finalizer is removed and the object
// disappears from the API:
//
//	Pass 1 — set topologyState = "destroying", leave the finalizer, requeue.
//	Pass 2 — state is already "destroying"; remove the finalizer so GC proceeds.
//
// The status write in pass 1 is skipped when the topology was never fully
// reconciled (status.kind is empty), to avoid persisting an otherwise-invalid
// status.
func (c *Controller) handleDeletion(
	ctx context.Context,
	req ctrlruntime.Request,
	topology *clabernetesapisv1alpha1.Topology,
) (ctrlruntime.Result, error) {
	if topology.Status.Kind != "" &&
		topology.Status.TopologyState != clabernetesconstants.TopologyStateDestroying {
		base := topology.DeepCopy()
		topology.Status.TopologyState = clabernetesconstants.TopologyStateDestroying

		err := c.BaseController.Client.Patch(ctx, topology, ctrlruntimeclient.MergeFrom(base))
		if err != nil {
			return ctrlruntime.Result{}, err
		}

		// Hold the state briefly so external watchers can observe the transition.
		return ctrlruntime.Result{RequeueAfter: destroyingStateObservabilityDelay}, nil
	}

	if !controllerutil.ContainsFinalizer(topology, clabernetesconstants.TopologyFinalizer) {
		return ctrlruntime.Result{}, nil
	}

	base := topology.DeepCopy()
	controllerutil.RemoveFinalizer(topology, clabernetesconstants.TopologyFinalizer)

	err := c.BaseController.Client.Patch(ctx, topology, ctrlruntimeclient.MergeFrom(base))
	if err != nil {
		// Finalizer removal failed. Re-fetch and mark as "destroyfailed" so
		// external watchers can observe the stuck state. Always return the
		// original error so the controller keeps retrying.
		fresh, getErr := c.getTopologyFromReq(ctx, req)
		if getErr == nil &&
			fresh.Status.TopologyState != clabernetesconstants.TopologyStateDestroyFailed {
			freshBase := fresh.DeepCopy()
			fresh.Status.TopologyState = clabernetesconstants.TopologyStateDestroyFailed

			_ = c.BaseController.Client.Patch(ctx, fresh, ctrlruntimeclient.MergeFrom(freshBase))
		}

		return ctrlruntime.Result{}, err
	}

	return ctrlruntime.Result{}, nil
}

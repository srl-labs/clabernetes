package containerlab

import (
	"context"

	clabernetesutil "github.com/srl-labs/clabernetes/util"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	"gopkg.in/yaml.v3"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// Reconcile handles reconciliation for this controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	clab, err := c.getClabFromReq(ctx, req)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			// was deleted, nothing to do
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if clab.DeletionTimestamp != nil {
		// deleting nothing to do, we have no finalizers or anything at this point
		return ctrlruntime.Result{}, nil
	}

	preReconcileConfigs := make(map[string]*clabernetesutilcontainerlab.Config)

	if clab.Status.Configs != "" {
		err = yaml.Unmarshal([]byte(clab.Status.Configs), &preReconcileConfigs)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed parsing unmarshalling previously stored config, error: %s", err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	// load the containerlab topo to make sure its all good
	containerlabTopo, err := clabernetesutilcontainerlab.LoadContainerlabTopology(clab.Spec.Config)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing containerlab config, error: ", err)

		return ctrlruntime.Result{}, err
	}

	clabernetesConfigs, tunnels, configShouldUpdate, err := c.processConfig(clab, containerlabTopo)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing containerlab config, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		clab,
		clabernetesConfigs,
		tunnels,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes config map, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		clab,
		preReconcileConfigs,
		clabernetesConfigs,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileServiceFabric(ctx, clab, clabernetesConfigs)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes services, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	var exposeServicesShouldUpdate bool

	if !clab.Spec.DisableExpose {
		exposeServicesShouldUpdate, err = c.TopologyReconciler.ReconcileServicesExpose(
			ctx,
			clab,
			clabernetesConfigs,
		)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed reconciling clabernetes expose services, error: %s", err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	if clabernetesutil.AnyBoolTrue(configShouldUpdate, exposeServicesShouldUpdate) {
		// we should update because config hash or something changed, so push update to the object
		err = c.BaseController.Client.Update(ctx, clab)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed updating object '%s/%s' error: %s",
				clab.Namespace,
				clab.Name,
				err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

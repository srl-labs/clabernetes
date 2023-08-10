package containerlab

import (
	"context"

	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"

	clabernetescontainerlab "gitlab.com/carlmontanari/clabernetes/containerlab"

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

	preReconcileConfigs := make(map[string]*clabernetescontainerlab.Config)

	if clab.Status.Configs != "" {
		err = yaml.Unmarshal([]byte(clab.Status.Configs), &preReconcileConfigs)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed parsing unmarshalling previously stored config, error: %s", err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	// load the clab topo to make sure its all good
	clabTopo, err := clabernetescontainerlab.LoadContainerlabTopology(clab.Spec.Config)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing containerlab config, error: ", err)

		return ctrlruntime.Result{}, err
	}

	configs, tunnels, configShouldUpdate, err := c.processConfig(clab, clabTopo)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing containerlab config, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.reconcileConfigMap(ctx, clab, configs, tunnels)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes config map, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.reconcileDeployments(ctx, clab, preReconcileConfigs, configs)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.reconcileServices(ctx, clab, configs)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes services, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	var exposeServicesShouldUpdate bool

	if !clab.Spec.DisableExpose {
		exposeServicesShouldUpdate, err = c.reconcileExposeServices(ctx, clab, configs)
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

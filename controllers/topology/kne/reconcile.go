package kne

import (
	"context"

	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"

	clabernetescontainerlab "gitlab.com/carlmontanari/clabernetes/containerlab"
	claberneteskne "gitlab.com/carlmontanari/clabernetes/kne"
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

	preReconcileConfigs := make(map[string]*clabernetescontainerlab.Config)

	if kne.Status.Configs != "" {
		err = yaml.Unmarshal([]byte(kne.Status.Configs), &preReconcileConfigs)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed parsing unmarshalling previously stored config, error: %s", err,
			)

			return ctrlruntime.Result{}, err
		}
	}

	// load the kne topo to make sure its all good
	kneTopo, err := claberneteskne.LoadKneTopology(kne.Spec.Topology)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing kne topology, error: ", err)

		return ctrlruntime.Result{}, err
	}

	clabernetesConfigs, tunnels, configShouldUpdate, err := c.processConfig(kne, kneTopo)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing kne topology, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileConfigMap(
		ctx,
		kne,
		clabernetesConfigs,
		tunnels,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes config map, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileDeployments(
		ctx,
		kne,
		preReconcileConfigs,
		clabernetesConfigs,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes deployments, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	err = c.TopologyReconciler.ReconcileServiceFabric(ctx, kne, clabernetesConfigs)
	if err != nil {
		c.BaseController.Log.Criticalf("failed reconciling clabernetes services, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	var exposeServicesShouldUpdate bool

	if !kne.Spec.DisableExpose {
		exposeServicesShouldUpdate, err = c.TopologyReconciler.ReconcileServicesExpose(
			ctx,
			kne,
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

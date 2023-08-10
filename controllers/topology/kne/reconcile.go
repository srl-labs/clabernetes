package kne

import (
	"context"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"

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

	configs, tunnels, configShouldUpdate, err := c.processConfig(kne, kneTopo)
	if err != nil {
		c.BaseController.Log.Criticalf("failed processing kne topology, error: %s", err)

		return ctrlruntime.Result{}, err
	}

	_, _ = configs, tunnels
	// TODO *things*

	// TODO temp to not piss off k8s w/ required field
	kne.Status.NodeExposedPorts = map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{}

	if clabernetesutil.AnyBoolTrue(configShouldUpdate) {
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

package kne

import (
	"context"
	"fmt"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	ctrlruntimebuilder "sigs.k8s.io/controller-runtime/pkg/builder"

	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"

	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
)

// NewController returns a new Controller.
func NewController(
	ctx context.Context,
	appName string,
	config *rest.Config,
	client ctrlruntimeclient.Client,
) clabernetescontrollers.Controller {
	baseController := clabernetescontrollers.NewBaseController(
		ctx,
		fmt.Sprintf(
			"%s-%s",
			clabernetesapistopology.Group,
			clabernetesapistopology.Kne,
		),
		appName,
		config,
		client,
	)

	c := &Controller{
		BaseController: baseController,
		TopologyReconciler: &clabernetescontrollerstopology.Reconciler{
			Log:          baseController.Log,
			Client:       baseController.Client,
			ResourceKind: clabernetesapistopology.Kne,
			ResourceLister: func(
				ctx context.Context,
				client ctrlruntimeclient.Client,
			) ([]ctrlruntimeclient.Object, error) {
				knes := &clabernetesapistopologyv1alpha1.KneList{}

				err := client.List(ctx, knes)
				if err != nil {
					return nil, err
				}

				var out []ctrlruntimeclient.Object

				for idx := range knes.Items {
					out = append(
						out,
						&knes.Items[idx],
					)
				}

				return out, nil
			},
			ConfigManagerGetter: clabernetesconfig.GetManager,
		},
	}

	return c
}

// Controller is the Containerlab topology controller object.
type Controller struct {
	*clabernetescontrollers.BaseController
	TopologyReconciler *clabernetescontrollerstopology.Reconciler
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrlruntime.Manager) error {
	c.BaseController.Log.Infof(
		"setting up %s controller with manager",
		clabernetesapistopology.Kne,
	)

	return ctrlruntime.NewControllerManagedBy(mgr).
		WithOptions(
			ctrlruntimecontroller.Options{
				MaxConcurrentReconciles: 1,
			},
		).
		For(&clabernetesapistopologyv1alpha1.Kne{}).
		// watch services so we can update the status of containerlab object with load balancer
		// address
		Watches(
			&k8scorev1.Service{},
			ctrlruntimehandler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&clabernetesapistopologyv1alpha1.Kne{},
				ctrlruntimehandler.OnlyControllerOwner(),
			),
		).
		// watch configmaps so we can react to global config changes; predicates ensure we only
		// watch the "clabernetes-config" (or appName-config) configmap
		Watches(
			&k8scorev1.ConfigMap{},
			ctrlruntimehandler.EnqueueRequestsFromMapFunc(
				c.TopologyReconciler.EnqueueForAll,
			),
			ctrlruntimebuilder.WithPredicates(
				c.BaseController.GlobalConfigPredicates(),
			),
		).
		Complete(c)
}

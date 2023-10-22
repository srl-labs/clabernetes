package containerlab

import (
	"context"
	"fmt"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimebuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
			clabernetesapistopology.Containerlab,
		),
		appName,
		config,
		client,
	)

	c := &Controller{
		BaseController: baseController,
		TopologyReconciler: clabernetescontrollerstopologyreconciler.NewReconciler(
			baseController.Log,
			baseController.Client,
			clabernetesapistopology.Containerlab,
			func(
				ctx context.Context,
				client ctrlruntimeclient.Client,
			) ([]ctrlruntimeclient.Object, error) {
				containerlabs := &clabernetesapistopologyv1alpha1.ContainerlabList{}

				err := client.List(ctx, containerlabs)
				if err != nil {
					return nil, err
				}

				var out []ctrlruntimeclient.Object

				for idx := range containerlabs.Items {
					out = append(
						out,
						&containerlabs.Items[idx],
					)
				}

				return out, nil
			},
			clabernetesconfig.GetManager,
		),
	}

	return c
}

// Controller is the Containerlab topology controller object.
type Controller struct {
	*clabernetescontrollers.BaseController
	TopologyReconciler *clabernetescontrollerstopologyreconciler.Reconciler
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrlruntime.Manager) error {
	c.BaseController.Log.Infof(
		"setting up %s controller with manager",
		clabernetesapistopology.Containerlab,
	)

	return ctrlruntime.NewControllerManagedBy(mgr).
		WithOptions(
			ctrlruntimecontroller.Options{
				MaxConcurrentReconciles: 1,
			},
		).
		For(&clabernetesapistopologyv1alpha1.Containerlab{}).
		// watch services so we can update the status of containerlab object with load balancer
		// address
		Watches(
			&k8scorev1.Service{},
			ctrlruntimehandler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&clabernetesapistopologyv1alpha1.Containerlab{},
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

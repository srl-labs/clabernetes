package containerlab

import (
	"context"
	"fmt"

	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"
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
		TopologyReconciler: &clabernetescontrollerstopology.Reconciler{
			Log:          baseController.Log,
			Client:       baseController.Client,
			ResourceKind: clabernetesapistopology.Containerlab,
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
			ctrlruntimehandler.EnqueueRequestsFromMapFunc(
				c.TopologyReconciler.MapServiceToContainerlab,
			),
		).
		Complete(c)
}

package kne

import (
	"context"
	"fmt"

	clabernetescontrollerstopology "gitlab.com/carlmontanari/clabernetes/controllers/topology"

	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
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
			Log:    baseController.Log,
			Client: baseController.Client,
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
		Complete(c)
}

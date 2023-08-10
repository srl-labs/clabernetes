package kne

import (
	"context"
	"fmt"

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
	c := &Controller{
		BaseController: clabernetescontrollers.NewBaseController(
			ctx,
			fmt.Sprintf(
				"%s-%s",
				clabernetesapistopology.Group,
				clabernetesapistopology.Kne,
			),
			appName,
			config,
			client,
		),
	}

	return c
}

// Controller is the Containerlab topology controller object.
type Controller struct {
	*clabernetescontrollers.BaseController
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

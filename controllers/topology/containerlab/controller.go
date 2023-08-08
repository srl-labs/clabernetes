package containerlab

import (
	"context"
	"fmt"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"
	ctrlruntimereconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
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
				clabernetesapistopology.Containerlab,
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
			ctrlruntimehandler.EnqueueRequestsFromMapFunc(c.mapServiceToContainerlab),
		).
		Complete(c)
}

func (c *Controller) mapServiceToContainerlab(
	_ context.Context,
	obj ctrlruntimeclient.Object,
) []ctrlruntimereconcile.Request {
	service, ok := obj.(*k8scorev1.Service)
	if !ok {
		c.BaseController.Log.Critical(
			"failed casting object to service in service to containerlab map func, " +
				"this should not happen. continuing but will not schedule any reconciles for this" +
				"service object",
		)

		return nil
	}

	labels := service.GetLabels()

	_, clabernetesOk := labels[clabernetesconstants.LabelApp]
	if !clabernetesOk {
		c.BaseController.Log.Debugf(
			"service '%s/%s' is not a clabernetes service, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	_, clabernetesExposeOk := labels[clabernetesconstants.LabelTopologyServiceType]
	if !clabernetesExposeOk {
		c.BaseController.Log.Debugf(
			"service '%s/%s' is not a clabernetes expose service, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	clabResource, clabResourceOk := labels[clabernetesconstants.LabelTopologyOwner]
	if !clabResourceOk {
		c.BaseController.Log.Criticalf(
			"service '%s/%s' is a clabernetes expose service, but cannot determine"+
				" corresponding clabernetes resource",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	c.BaseController.Log.Infof(
		"service '%s/%s' is clabernetes expose service,"+
			" scheduling reconcile for containerlab resource '%s/%s' ",
		service.Namespace,
		service.Name,
		service.Namespace,
		clabResource,
	)

	return []ctrlruntimereconcile.Request{
		{
			NamespacedName: apimachinerytypes.NamespacedName{
				Namespace: service.Namespace,
				Name:      clabResource,
			},
		},
	}
}

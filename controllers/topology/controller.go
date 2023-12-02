package topology

import (
	"context"

	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimereconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"

	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimebuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"

	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// NewController returns a new Controller.
func NewController(
	clabernetes clabernetesmanagertypes.Clabernetes,
) clabernetescontrollers.Controller {
	ctx := clabernetes.GetContext()

	baseController := clabernetescontrollers.NewBaseController(
		ctx,
		clabernetesapis.Topology,
		clabernetes.GetAppName(),
		clabernetes.GetKubeConfig(),
		clabernetes.GetCtrlRuntimeClient(),
	)

	c := &Controller{
		BaseController: baseController,
		TopologyReconciler: NewReconciler(
			baseController.Log,
			baseController.Client,
			clabernetes.GetAppName(),
			clabernetes.GetNamespace(),
			clabernetesconfig.GetManager,
			clabernetes.GetClusterCRIKind(),
			clabernetesutil.GetEnvStrOrDefault(
				clabernetesconstants.LauncherImagePullThroughModeEnv,
				clabernetesconstants.LauncherDefaultImagePullThroughMode,
			),
		),
	}

	return c
}

// Controller is the Containerlab topology controller object.
type Controller struct {
	*clabernetescontrollers.BaseController
	TopologyReconciler *Reconciler
}

// enqueueForAll enqueues all Topology CRs for reconciliation.
func (c *Controller) enqueueForAll(
	ctx context.Context,
	_ ctrlruntimeclient.Object,
) []ctrlruntimereconcile.Request {
	topologies := &clabernetesapisv1alpha1.TopologyList{}

	err := c.Client.List(ctx, topologies)
	if err != nil {
		c.Log.Criticalf("failed listing resource objects in EnqueueForAll, err: %s", err)

		return nil
	}

	requests := make([]ctrlruntimereconcile.Request, len(topologies.Items))

	for idx := range topologies.Items {
		requests[idx] = ctrlruntimereconcile.Request{
			NamespacedName: apimachinerytypes.NamespacedName{
				Namespace: topologies.Items[idx].GetNamespace(),
				Name:      topologies.Items[idx].GetName(),
			},
		}
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrlruntime.Manager) error {
	c.BaseController.Log.Infof(
		"setting up %s controller with manager",
		clabernetesapis.Topology,
	)

	return ctrlruntime.NewControllerManagedBy(mgr).
		WithOptions(
			ctrlruntimecontroller.Options{
				MaxConcurrentReconciles: 1,
			},
		).
		For(&clabernetesapisv1alpha1.Topology{}).
		// watch services so we can update the status of containerlab object with load balancer
		// address
		Watches(
			&k8scorev1.Service{},
			ctrlruntimehandler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&clabernetesapisv1alpha1.Topology{},
				ctrlruntimehandler.OnlyControllerOwner(),
			),
		).
		// watch configmaps so we can react to global config changes; predicates ensure we only
		// watch the "clabernetes-config" (or appName-config) configmap
		Watches(
			&k8scorev1.ConfigMap{},
			ctrlruntimehandler.EnqueueRequestsFromMapFunc(
				c.enqueueForAll,
			),
			ctrlruntimebuilder.WithPredicates(
				c.BaseController.GlobalConfigPredicates(),
			),
		).
		Complete(c)
}

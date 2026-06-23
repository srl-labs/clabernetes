package node

import (
	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
	k8sappsv1 "k8s.io/api/apps/v1"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"
)

// concurrentReconciles is the number of Node objects we are willing to reconcile in parallel. Node
// reconciles are bounded, independent units of work (one node's ConfigMap/Deployment/Service/PVC),
// so -- unlike the Topology controller which serializes at 1 -- we can safely fan these out.
const concurrentReconciles = 25

// Controller is the clabernetes Node controller. It reconciles the per-node resources for a single
// Node custom resource (the decomposed unit of a Topology); see
// docs/design/0001-scale-node-link-crds.md.
type Controller struct {
	*clabernetescontrollers.BaseController

	NodeReconciler *Reconciler
}

// NewController returns a new Node Controller.
func NewController(
	clabernetes clabernetesmanagertypes.Clabernetes,
) clabernetescontrollers.Controller {
	ctx := clabernetes.GetContext()

	baseController := clabernetescontrollers.NewBaseController(
		ctx,
		clabernetesapis.Node,
		clabernetes.GetAppName(),
		clabernetes.GetKubeConfig(),
		clabernetes.GetCtrlRuntimeClient(),
	)

	c := &Controller{
		BaseController: baseController,
		NodeReconciler: NewReconciler(
			baseController.Log,
			baseController.Client,
			clabernetes.GetAppName(),
			clabernetes.GetNamespace(),
			clabernetes.GetClusterCRIKind(),
			clabernetesconfig.GetManager,
		),
	}

	return c
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrlruntime.Manager) error {
	c.BaseController.Log.Infof(
		"setting up %s controller with manager",
		clabernetesapis.Node,
	)

	return ctrlruntime.NewControllerManagedBy(mgr).
		WithOptions(
			ctrlruntimecontroller.Options{
				MaxConcurrentReconciles: concurrentReconciles,
			},
		).
		For(&clabernetesapisv1alpha1.Node{}).
		// watch the owned Deployment so that when the launcher pod becomes (un)available the Node is
		// re-reconciled and its status.ready is refreshed from the Deployment's Available condition.
		// without this the Node only reconciles on spec changes and its readiness goes stale.
		Watches(
			&k8sappsv1.Deployment{},
			ctrlruntimehandler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&clabernetesapisv1alpha1.Node{},
			),
		).
		Complete(c)
}

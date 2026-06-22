package node

import (
	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
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
		Complete(c)
}

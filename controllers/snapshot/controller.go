package snapshot

import (
	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
	"k8s.io/client-go/kubernetes"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
)

// NewController returns a new snapshot Controller.
func NewController(
	clabernetes clabernetesmanagertypes.Clabernetes,
) clabernetescontrollers.Controller {
	ctx := clabernetes.GetContext()

	baseController := clabernetescontrollers.NewBaseController(
		ctx,
		clabernetesapis.Snapshot,
		clabernetes.GetAppName(),
		clabernetes.GetKubeConfig(),
		clabernetes.GetCtrlRuntimeClient(),
	)

	c := &Controller{
		BaseController: baseController,
		KubeClient:     clabernetes.GetKubeClient(),
	}

	return c
}

// Controller is the Snapshot controller object.
type Controller struct {
	*clabernetescontrollers.BaseController

	// KubeClient is the standard kubernetes client used for pod exec operations.
	KubeClient *kubernetes.Clientset
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrlruntime.Manager) error {
	c.BaseController.Log.Infof(
		"setting up %s controller with manager",
		clabernetesapis.Snapshot,
	)

	return ctrlruntime.NewControllerManagedBy(mgr).
		WithOptions(
			ctrlruntimecontroller.Options{
				MaxConcurrentReconciles: 1,
			},
		).
		For(&clabernetesapisv1alpha1.Snapshot{}).
		Complete(c)
}

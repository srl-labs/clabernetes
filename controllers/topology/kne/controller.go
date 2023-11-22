package kne

import (
	"context"
	"fmt"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutil "github.com/srl-labs/clabernetes/util"

	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"

	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	ctrlruntimebuilder "sigs.k8s.io/controller-runtime/pkg/builder"

	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimehandler "sigs.k8s.io/controller-runtime/pkg/handler"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
)

// NewController returns a new Controller.
func NewController(
	clabernetes clabernetesmanagertypes.Clabernetes,
) clabernetescontrollers.Controller {
	ctx := clabernetes.GetContext()

	baseController := clabernetescontrollers.NewBaseController(
		ctx,
		fmt.Sprintf(
			"%s-%s",
			clabernetesapistopology.Group,
			clabernetesapistopology.Kne,
		),
		clabernetes.GetAppName(),
		clabernetes.GetKubeConfig(),
		clabernetes.GetCtrlRuntimeClient(),
	)

	c := &Controller{
		BaseController: baseController,
		TopologyReconciler: clabernetescontrollerstopologyreconciler.NewReconciler(
			baseController.Log,
			baseController.Client,
			clabernetes.GetAppName(),
			clabernetes.GetNamespace(),
			clabernetesapistopology.Kne,
			func(
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
	TopologyReconciler *clabernetescontrollerstopologyreconciler.Reconciler
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

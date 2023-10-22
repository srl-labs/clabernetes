package reconciler

import (
	"context"
	"fmt"
	"slices"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8sappsv1 "k8s.io/api/apps/v1"

	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimereconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ResourceListerFunc represents a function that can list the objects that a topology controller
// is responsible for.
type ResourceListerFunc func(
	ctx context.Context,
	client ctrlruntimeclient.Client,
) ([]ctrlruntimeclient.Object, error)

// NewReconciler creates a new generic Reconciler (TopologyReconciler).
func NewReconciler(
	log claberneteslogging.Instance,
	client ctrlruntimeclient.Client,
	owningTopologyKind string,
	resourceLister ResourceListerFunc,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *Reconciler {
	return &Reconciler{
		Log:            log,
		Client:         client,
		ResourceKind:   owningTopologyKind,
		ResourceLister: resourceLister,

		configMapReconciler: NewConfigMapReconciler(
			log,
			owningTopologyKind,
			configManagerGetter,
		),
		deploymentReconciler: NewDeploymentReconciler(
			log,
			owningTopologyKind,
			configManagerGetter,
		),
		serviceFabricReconciler: NewServiceFabricReconciler(
			log,
			owningTopologyKind,
			configManagerGetter,
		),
		serviceExposeReconciler: NewServiceExposeReconciler(
			log,
			owningTopologyKind,
			configManagerGetter,
		),
	}
}

// Reconciler (TopologyReconciler) is the base clabernetes topology reconciler that is embedded in
// all clabernetes topology controllers, it provides common methods for reconciling the
// common/standard resources that represent a clabernetes object (configmap, deployments,
// services, etc.).
type Reconciler struct {
	Log            claberneteslogging.Instance
	Client         ctrlruntimeclient.Client
	ResourceKind   string
	ResourceLister ResourceListerFunc

	configMapReconciler     *ConfigMapReconciler
	deploymentReconciler    *DeploymentReconciler
	serviceFabricReconciler *ServiceFabricReconciler
	serviceExposeReconciler *ServiceExposeReconciler
}

// ReconcileConfigMap reconciles the primary configmap containing clabernetes configs and tunnel
// information.
func (r *Reconciler) ReconcileConfigMap(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) error {
	namespacedName := apimachinerytypes.NamespacedName{
		Namespace: owningTopology.GetNamespace(),
		Name:      owningTopology.GetName(),
	}

	renderedConfigMap, err := r.configMapReconciler.Render(
		namespacedName,
		clabernetesConfigs,
		tunnels,
	)
	if err != nil {
		return err
	}

	existingConfigMap := &k8scorev1.ConfigMap{}

	err = r.Client.Get(
		ctx,
		namespacedName,
		existingConfigMap,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.createObj(
				ctx,
				owningTopology,
				renderedConfigMap,
				clabernetesconstants.KubernetesConfigMap,
			)
		}

		return err
	}

	if r.configMapReconciler.Conforms(
		existingConfigMap,
		renderedConfigMap,
		owningTopology.GetUID(),
	) {
		return nil
	}

	return r.updateObj(ctx, renderedConfigMap, clabernetesconstants.KubernetesConfigMap)
}

func (r *Reconciler) reconcileDeploymentsHandleRestarts(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	previousClabernetesConfigs,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	deployments *clabernetesutil.ObjectDiffer[*k8sappsv1.Deployment],
) error {
	r.Log.Info("determining nodes needing restart")

	nodesNeedingRestart := r.deploymentReconciler.DetermineNodesNeedingRestart(
		previousClabernetesConfigs,
		currentClabernetesConfigs,
	)
	if len(nodesNeedingRestart) == 0 {
		return nil
	}

	for _, nodeName := range nodesNeedingRestart {
		if slices.Contains(deployments.Missing, nodeName) {
			// is a new node, don't restart, we'll deploy it soon
			continue
		}

		r.Log.Infof(
			"restarting the nodes '%s' as configurations have changed",
			nodesNeedingRestart,
		)

		deploymentName := fmt.Sprintf("%s-%s", owningTopology.GetName(), nodeName)

		nodeDeployment := &k8sappsv1.Deployment{}

		err := r.getObj(
			ctx,
			nodeDeployment,
			apimachinerytypes.NamespacedName{
				Namespace: owningTopology.GetNamespace(),
				Name:      deploymentName,
			},
			clabernetesconstants.KubernetesDeployment,
		)
		if err != nil {
			if apimachineryerrors.IsNotFound(err) {
				r.Log.Warnf(
					"could not find deployment '%s', cannot restart after config change,"+
						" this should not happen",
					deploymentName,
				)

				return nil
			}

			return err
		}

		if nodeDeployment.Spec.Template.ObjectMeta.Annotations == nil {
			nodeDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}

		now := time.Now().Format(time.RFC3339)

		nodeDeployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = now //nolint:lll

		err = r.updateObj(ctx, nodeDeployment, clabernetesconstants.KubernetesDeployment)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileDeployments reconciles the deployments that make up a clabernetes Topology.
func (r *Reconciler) ReconcileDeployments(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	previousClabernetesConfigs,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) error {
	deployments, err := reconcileResolve(
		ctx,
		r,
		&k8sappsv1.Deployment{},
		&k8sappsv1.DeploymentList{},
		clabernetesconstants.KubernetesDeployment,
		owningTopology,
		currentClabernetesConfigs,
		r.deploymentReconciler.Resolve,
	)
	if err != nil {
		return err
	}

	r.Log.Info("pruning extraneous deployments")

	for _, extraDeployment := range deployments.Extra {
		err = r.deleteObj(ctx, extraDeployment, clabernetesconstants.KubernetesDeployment)
		if err != nil {
			return err
		}
	}

	r.Log.Info("creating missing deployments")

	renderedMissingDeployments := r.deploymentReconciler.RenderAll(
		owningTopology,
		currentClabernetesConfigs,
		deployments.Missing,
	)

	for _, renderedMissingDeployment := range renderedMissingDeployments {
		err = r.createObj(
			ctx,
			owningTopology,
			renderedMissingDeployment,
			clabernetesconstants.KubernetesDeployment,
		)
		if err != nil {
			return err
		}
	}

	r.Log.Info("enforcing desired state on existing deployments")

	for existingCurrentDeploymentNodeName, existingCurrentDeployment := range deployments.Current {
		renderedCurrentDeployment := r.deploymentReconciler.Render(
			owningTopology,
			currentClabernetesConfigs,
			existingCurrentDeploymentNodeName,
		)

		err = ctrlruntimeutil.SetOwnerReference(
			owningTopology,
			renderedCurrentDeployment,
			r.Client.Scheme(),
		)
		if err != nil {
			return err
		}

		if !r.deploymentReconciler.Conforms(
			existingCurrentDeployment,
			renderedCurrentDeployment,
			owningTopology.GetUID(),
		) {
			err = r.updateObj(
				ctx,
				renderedCurrentDeployment,
				clabernetesconstants.KubernetesDeployment,
			)
			if err != nil {
				return err
			}
		}
	}

	return r.reconcileDeploymentsHandleRestarts(
		ctx,
		owningTopology,
		previousClabernetesConfigs,
		currentClabernetesConfigs,
		deployments,
	)
}

// ReconcileServiceFabric reconciles the service used for "fabric" (inter node) connectivity.
func (r *Reconciler) ReconcileServiceFabric(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) error {
	serviceTypeName := fmt.Sprintf("fabric %s", clabernetesconstants.KubernetesService)

	services, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.Service{},
		&k8scorev1.ServiceList{},
		serviceTypeName,
		owningTopology,
		currentClabernetesConfigs,
		r.serviceExposeReconciler.Resolve,
	)
	if err != nil {
		return err
	}

	r.Log.Info("pruning extraneous fabric services")

	for _, extraService := range services.Extra {
		err = r.deleteObj(
			ctx,
			extraService,
			serviceTypeName,
		)
		if err != nil {
			return err
		}
	}

	r.Log.Info("creating missing fabric services")

	renderedMissingServices := r.serviceFabricReconciler.RenderAll(
		owningTopology,
		services.Missing,
	)

	for _, renderedMissingService := range renderedMissingServices {
		err = r.createObj(
			ctx,
			owningTopology,
			renderedMissingService,
			serviceTypeName,
		)
		if err != nil {
			return err
		}
	}

	r.Log.Info("enforcing desired state on fabric services")

	for existingCurrentServiceNodeName, existingCurrentService := range services.Current {
		renderedCurrentService := r.serviceFabricReconciler.Render(
			owningTopology,
			existingCurrentServiceNodeName,
		)

		err = ctrlruntimeutil.SetOwnerReference(
			owningTopology,
			renderedCurrentService,
			r.Client.Scheme(),
		)
		if err != nil {
			return err
		}

		if !r.serviceFabricReconciler.Conforms(
			existingCurrentService,
			renderedCurrentService,
			owningTopology.GetUID(),
		) {
			err = r.updateObj(
				ctx,
				renderedCurrentService,
				serviceTypeName,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ReconcileServicesExpose reconciles the service(s) used for exposing nodes.
func (r *Reconciler) ReconcileServicesExpose(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) (bool, error) {
	serviceTypeName := fmt.Sprintf("expose %s", clabernetesconstants.KubernetesService)

	var shouldUpdate bool

	owningTopologyStatus := owningTopology.GetTopologyStatus()

	if owningTopologyStatus.NodeExposedPorts == nil {
		owningTopologyStatus.NodeExposedPorts = map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{} //nolint:lll

		shouldUpdate = true
	}

	services, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.Service{},
		&k8scorev1.ServiceList{},
		serviceTypeName,
		owningTopology,
		currentClabernetesConfigs,
		r.serviceExposeReconciler.Resolve,
	)
	if err != nil {
		return shouldUpdate, err
	}

	r.Log.Info("pruning extraneous services")

	for _, extraDeployment := range services.Extra {
		err = r.deleteObj(
			ctx,
			extraDeployment,
			serviceTypeName,
		)
		if err != nil {
			return shouldUpdate, err
		}
	}

	r.Log.Info("creating missing services")

	renderedMissingServices := r.serviceExposeReconciler.RenderAll(
		owningTopology,
		&owningTopologyStatus,
		currentClabernetesConfigs,
		services.Missing,
	)

	for _, renderedMissingService := range renderedMissingServices {
		err = r.createObj(
			ctx,
			owningTopology,
			renderedMissingService,
			serviceTypeName,
		)
		if err != nil {
			return shouldUpdate, err
		}
	}

	for existingCurrentServiceNodeName, existingCurrentService := range services.Current {
		renderedCurrentService := r.serviceExposeReconciler.Render(
			owningTopology,
			&owningTopologyStatus,
			currentClabernetesConfigs,
			existingCurrentServiceNodeName,
		)

		if len(existingCurrentService.Status.LoadBalancer.Ingress) == 1 {
			// can/would this ever be more than 1? i dunno?
			address := existingCurrentService.Status.LoadBalancer.Ingress[0].IP
			if address != "" {
				owningTopologyStatus.NodeExposedPorts[existingCurrentServiceNodeName].LoadBalancerAddress = address //nolint:lll
			}
		}

		err = ctrlruntimeutil.SetOwnerReference(
			owningTopology,
			renderedCurrentService,
			r.Client.Scheme(),
		)
		if err != nil {
			return shouldUpdate, err
		}

		if !r.serviceExposeReconciler.Conforms(
			existingCurrentService,
			renderedCurrentService,
			owningTopology.GetUID(),
		) {
			err = r.updateObj(
				ctx,
				renderedCurrentService,
				serviceTypeName,
			)
			if err != nil {
				return shouldUpdate, err
			}
		}
	}

	nodeExposedPortsBytes, err := yaml.Marshal(owningTopologyStatus.NodeExposedPorts)
	if err != nil {
		return shouldUpdate, err
	}

	newNodeExposedPortsHash := clabernetesutil.HashBytes(nodeExposedPortsBytes)

	if owningTopologyStatus.NodeExposedPortsHash != newNodeExposedPortsHash {
		owningTopologyStatus.NodeExposedPortsHash = newNodeExposedPortsHash

		owningTopology.SetTopologyStatus(owningTopologyStatus)

		shouldUpdate = true
	}

	return shouldUpdate, nil
}

// EnqueueForAll enqueues a reconcile for kinds the Reconciler represents. This is probably not very
// efficient/good but we should have low volume and we're using the cached ctrlruntime client so its
// probably ok :).
func (r *Reconciler) EnqueueForAll(
	ctx context.Context,
	_ ctrlruntimeclient.Object,
) []ctrlruntimereconcile.Request {
	objList, err := r.ResourceLister(ctx, r.Client)
	if err != nil {
		r.Log.Criticalf("failed listing resource objects in EnqueueForAll, err: %s", err)

		return nil
	}

	requests := make([]ctrlruntimereconcile.Request, len(objList))

	for idx := range objList {
		requests[idx] = ctrlruntimereconcile.Request{
			NamespacedName: apimachinerytypes.NamespacedName{
				Namespace: objList[idx].GetNamespace(),
				Name:      objList[idx].GetName(),
			},
		}
	}

	return requests
}

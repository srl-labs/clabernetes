package reconciler

import (
	"context"
	"fmt"
	"slices"
	"time"

	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8sappsv1 "k8s.io/api/apps/v1"

	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
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
	managerAppName,
	managerNamespace,
	owningTopologyKind string,
	resourceLister ResourceListerFunc,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
	criKind,
	imagePullThroughMode string,
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
		serviceNodeAliasReconciler: NewServiceNodeAliasReconciler(
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
		deploymentReconciler: NewDeploymentReconciler(
			log,
			managerAppName,
			managerNamespace,
			owningTopologyKind,
			configManagerGetter,
			criKind,
			imagePullThroughMode,
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

	configMapReconciler        *ConfigMapReconciler
	serviceNodeAliasReconciler *ServiceNodeAliasReconciler
	serviceFabricReconciler    *ServiceFabricReconciler
	serviceExposeReconciler    *ServiceExposeReconciler
	deploymentReconciler       *DeploymentReconciler
}

// ReconcileConfigMap reconciles the primary configmap containing clabernetes configs, tunnel
// information, pull secret information, and perhaps more in the future.
func (r *Reconciler) ReconcileConfigMap(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	reconcileData *ReconcileData,
) error {
	var err error

	topologyCommonSpec := owningTopology.GetTopologyCommonSpec()

	configBytes, configHash, err := clabernetesutil.HashObjectYAML(
		reconcileData.ResolvedConfigs,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedConfigsBytes = configBytes
	reconcileData.ResolvedConfigsHash = configHash

	_, tunnelHash, err := clabernetesutil.HashObjectYAML(
		reconcileData.ResolvedConfigs,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedTunnelsHash = tunnelHash

	for nodeName, nodeFilesFromURL := range topologyCommonSpec.FilesFromURL {
		var nodeFilesFromURLHash string

		_, nodeFilesFromURLHash, err = clabernetesutil.HashObject(nodeFilesFromURL)
		if err != nil {
			return err
		}

		reconcileData.ResolvedFilesFromURLHashes[nodeName] = nodeFilesFromURLHash

		if reconcileData.PreviousFilesFromURLHashes[nodeName] != nodeFilesFromURLHash {
			// files from url hash has changed, need to smack the node so the configmap update
			// gets realized
			reconcileData.NodesNeedingReboot.Add(nodeName)
		}
	}

	imagePullSecretsBytes, imagePullSecretsHash, err := clabernetesutil.HashObjectYAML(
		topologyCommonSpec.ImagePullSecrets,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedImagePullSecretsHash = imagePullSecretsHash

	if !reconcileData.ConfigMapHasChanges() {
		// the configs hashes match, nothing to do, should reconcile is false, and no error, *but*
		// because the services may force us to update the cr we are reconciling, and we haven't
		// processed the tunnel ids yet (because its slow and we are lazy), we need to copy the
		// *previous* tunnel data into our "current" tunnel data so we make sure to not update the
		// cr status with tunnel data with all zero for the tunnel ids
		reconcileData.ResolvedTunnels = reconcileData.PreviousTunnels

		return nil
	}

	clabernetescontrollerstopology.AllocateTunnelIDs(
		reconcileData.PreviousTunnels,
		reconcileData.ResolvedTunnels,
	)

	// we need to tell the controller to update the originating CR because obviously our hashes
	// dont match which means we had some changes from the previous reconcile
	reconcileData.ShouldUpdateResource = true

	namespacedName := apimachinerytypes.NamespacedName{
		Namespace: owningTopology.GetNamespace(),
		Name:      owningTopology.GetName(),
	}

	renderedConfigMap, err := r.configMapReconciler.Render(
		namespacedName,
		reconcileData.ResolvedConfigs,
		reconcileData.ResolvedTunnels,
		topologyCommonSpec.FilesFromURL,
		string(imagePullSecretsBytes),
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
	deployments *clabernetesutil.ObjectDiffer[*k8sappsv1.Deployment],
	reconcileData *ReconcileData,
) error {
	r.Log.Debug("determining nodes needing restart")

	r.deploymentReconciler.DetermineNodesNeedingRestart(
		reconcileData,
	)

	if reconcileData.NodesNeedingReboot.Len() == 0 {
		r.Log.Debug("all nodes are up to date, no restarts required")

		return nil
	}

	for _, nodeName := range reconcileData.NodesNeedingReboot.Items() {
		if slices.Contains(deployments.Missing, nodeName) {
			// is a new node, don't restart, we'll deploy it soon
			continue
		}

		r.Log.Infof(
			"restarting the node '%s' as configurations have changed",
			nodeName,
		)

		if r.Log.GetLevel() == clabernetesconstants.Debug {
			diff, err := clabernetesutil.UnifiedDiff(
				reconcileData.ResolvedConfigs[nodeName],
				reconcileData.PreviousConfigs[nodeName],
			)
			if err != nil {
				r.Log.Warnf(
					"failed generating diff of deployment. this only happened because logging"+
						" is at debug level, ignoring the error. err: %s",
					err,
				)
			} else {
				r.Log.Debugf("deployment diff: %s", diff)
			}
		}

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
	reconcileData *ReconcileData,
) error {
	deployments, err := reconcileResolve(
		ctx,
		r,
		&k8sappsv1.Deployment{},
		&k8sappsv1.DeploymentList{},
		clabernetesconstants.KubernetesDeployment,
		owningTopology,
		reconcileData.ResolvedConfigs,
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
		reconcileData.ResolvedConfigs,
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
			reconcileData.ResolvedConfigs,
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
		deployments,
		reconcileData,
	)
}

// ReconcileServices reconciles all the services for a clabernetes Topology.
func (r *Reconciler) ReconcileServices(
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	reconcileData *ReconcileData,
) error {
	err := r.ReconcileServiceNodeAlias(
		ctx,
		owningTopology,
		reconcileData,
	)
	if err != nil {
		r.Log.Criticalf(
			"failed reconciling clabernetes node alias services, error: %s", err,
		)

		return err
	}

	err = r.ReconcileServiceFabric(
		ctx,
		owningTopology,
		reconcileData,
	)
	if err != nil {
		r.Log.Criticalf(
			"failed reconciling clabernetes fabric services, error: %s", err,
		)

		return err
	}

	err = r.ReconcileServicesExpose(
		ctx,
		owningTopology,
		reconcileData,
	)
	if err != nil {
		r.Log.Criticalf(
			"failed reconciling clabernetes expose services, error: %s", err,
		)

		return err
	}

	return nil
}

// ReconcileServiceNodeAlias reconciles the service used for "node alias" -- that is,
// making it so that resolution in c9s is more or less the same as with "normal" containerlab in
// docker.
func (r *Reconciler) ReconcileServiceNodeAlias( //nolint: dupl
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	reconcileData *ReconcileData,
) error {
	serviceTypeName := fmt.Sprintf("nodeAlias %s", clabernetesconstants.KubernetesService)

	services, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.Service{},
		&k8scorev1.ServiceList{},
		serviceTypeName,
		owningTopology,
		reconcileData.ResolvedConfigs,
		r.serviceNodeAliasReconciler.Resolve,
	)
	if err != nil {
		return err
	}

	r.Log.Info("pruning extraneous nodeAlias services")

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

	r.Log.Info("creating missing nodeAlias services")

	renderedMissingServices := r.serviceNodeAliasReconciler.RenderAll(
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

	r.Log.Info("enforcing desired state on nodeAlias services")

	for existingCurrentServiceNodeName, existingCurrentService := range services.Current {
		renderedCurrentService := r.serviceNodeAliasReconciler.Render(
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

		if !r.serviceNodeAliasReconciler.Conforms(
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

// ReconcileServiceFabric reconciles the service used for "fabric" (inter node) connectivity.
func (r *Reconciler) ReconcileServiceFabric( //nolint: dupl
	ctx context.Context,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	reconcileData *ReconcileData,
) error {
	serviceTypeName := fmt.Sprintf("fabric %s", clabernetesconstants.KubernetesService)

	services, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.Service{},
		&k8scorev1.ServiceList{},
		serviceTypeName,
		owningTopology,
		reconcileData.ResolvedConfigs,
		r.serviceFabricReconciler.Resolve,
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
	reconcileData *ReconcileData,
) error {
	serviceTypeName := fmt.Sprintf("expose %s", clabernetesconstants.KubernetesService)

	owningTopologyStatus := owningTopology.GetTopologyStatus()

	if owningTopologyStatus.NodeExposedPorts == nil {
		owningTopologyStatus.NodeExposedPorts = map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{} //nolint:lll

		// shouldUpdate is true because we didn't have any previously stored node exposed port
		// status data
		reconcileData.ShouldUpdateResource = true
	}

	services, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.Service{},
		&k8scorev1.ServiceList{},
		serviceTypeName,
		owningTopology,
		reconcileData.ResolvedConfigs,
		r.serviceExposeReconciler.Resolve,
	)
	if err != nil {
		return err
	}

	r.Log.Info("pruning extraneous services")

	for _, extraDeployment := range services.Extra {
		err = r.deleteObj(
			ctx,
			extraDeployment,
			serviceTypeName,
		)
		if err != nil {
			return err
		}
	}

	r.Log.Info("creating missing services")

	renderedMissingServices := r.serviceExposeReconciler.RenderAll(
		owningTopology,
		reconcileData,
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

	r.Log.Info("enforcing desired state on expose services")

	for existingCurrentServiceNodeName, existingCurrentService := range services.Current {
		renderedCurrentService := r.serviceExposeReconciler.Render(
			owningTopology,
			reconcileData,
			existingCurrentServiceNodeName,
		)

		if len(existingCurrentService.Status.LoadBalancer.Ingress) == 1 {
			// can/would this ever be more than 1? i dunno?
			address := existingCurrentService.Status.LoadBalancer.Ingress[0].IP
			if address != "" {
				reconcileData.ResolvedNodeExposedPorts[existingCurrentServiceNodeName].LoadBalancerAddress = address //nolint:lll
			}
		}

		err = ctrlruntimeutil.SetOwnerReference(
			owningTopology,
			renderedCurrentService,
			r.Client.Scheme(),
		)
		if err != nil {
			return err
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
				return err
			}
		}
	}

	_, newNodeExposedPortsHash, err := clabernetesutil.HashObject(
		owningTopologyStatus.NodeExposedPorts,
	)
	if err != nil {
		return err
	}

	if owningTopologyStatus.NodeExposedPortsHash != newNodeExposedPortsHash {
		reconcileData.ResolvedNodeExposedPortsHash = newNodeExposedPortsHash

		// our exposed hash stuff changed, we need to update the cr status
		reconcileData.ShouldUpdateResource = true
	}

	return nil
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

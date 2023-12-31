package topology

import (
	"context"
	"fmt"
	"slices"
	"time"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewReconciler creates a new generic Reconciler (TopologyReconciler).
func NewReconciler(
	log claberneteslogging.Instance,
	client ctrlruntimeclient.Client,
	managerAppName,
	managerNamespace,
	criKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *Reconciler {
	return &Reconciler{
		Log:    log,
		Client: client,
		serviceAccountReconciler: NewServiceAccountReconciler(
			log,
			client,
			configManagerGetter,
		),
		roleBindingReconciler: NewRoleBindingReconciler(
			log,
			client,
			configManagerGetter,
			managerAppName,
		),
		configMapReconciler: NewConfigMapReconciler(
			log,
			configManagerGetter,
		),
		serviceNodeAliasReconciler: NewServiceNodeAliasReconciler(
			log,
			configManagerGetter,
		),
		serviceFabricReconciler: NewServiceFabricReconciler(
			log,
			configManagerGetter,
		),
		serviceExposeReconciler: NewServiceExposeReconciler(
			log,
			configManagerGetter,
		),
		persistentVolumeClaimReconciler: NewPersistentVolumeClaimReconciler(
			log,
			configManagerGetter,
		),
		deploymentReconciler: NewDeploymentReconciler(
			log,
			managerAppName,
			managerNamespace,
			criKind,
			configManagerGetter,
		),
	}
}

// Reconciler (TopologyReconciler) is the base clabernetes topology reconciler that is embedded in
// all clabernetes topology controllers, it provides common methods for reconciling the
// common/standard resources that represent a clabernetes object (configmap, deployments,
// services, etc.).
type Reconciler struct {
	Log    claberneteslogging.Instance
	Client ctrlruntimeclient.Client

	serviceAccountReconciler        *ServiceAccountReconciler
	roleBindingReconciler           *RoleBindingReconciler
	configMapReconciler             *ConfigMapReconciler
	serviceNodeAliasReconciler      *ServiceNodeAliasReconciler
	serviceFabricReconciler         *ServiceFabricReconciler
	serviceExposeReconciler         *ServiceExposeReconciler
	persistentVolumeClaimReconciler *PersistentVolumeClaimReconciler
	deploymentReconciler            *DeploymentReconciler
}

// ReconcileNamespaceResources reconciles resources that exist in a Topology's namespace but are not
// 1:1 with a Topology -- for example ServiceAccount and RoleBinding resources which are created at
// the point the first Topology in a namespace is created and exist until the final Topology in a
// namespace is being removed.
func (r *Reconciler) ReconcileNamespaceResources(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
) error {
	err := r.ReconcileServiceAccount(ctx, owningTopology)
	if err != nil {
		return err
	}

	err = r.ReconcileRoleBinding(ctx, owningTopology)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileServiceAccount reconciles the service account for the given namespace -- note that there
// is only *one* service account per namespace, but its simply reconciled each time a Topology is
// reconciled to make life easy. This and the RoleBinding are the only resources we need to worry
// about when deleting, a Topology resource, hence there is `deleting` arg to indicate if we should
// see if we should clean things up.
func (r *Reconciler) ReconcileServiceAccount(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
) error {
	return r.serviceAccountReconciler.Reconcile(ctx, owningTopology)
}

// ReconcileRoleBinding reconciles the role binding for the given namespace -- note that there
// is only *one* role binding per namespace, but its simply reconciled each time a Topology is
// reconciled to make life easy. This and the ServiceAccount are the only resources we need to worry
// about when deleting, a Topology resource, hence there is `deleting` arg to indicate if we should
// see if we should clean things up.
func (r *Reconciler) ReconcileRoleBinding(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
) error {
	return r.roleBindingReconciler.Reconcile(ctx, owningTopology)
}

// ReconcileConfigMap reconciles the primary configmap containing clabernetes configs, tunnel
// information, pull secret information, and perhaps more in the future.
func (r *Reconciler) ReconcileConfigMap(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	var err error

	configBytes, configHash, err := clabernetesutil.HashObjectYAML(
		reconcileData.ResolvedConfigs,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedConfigsBytes = configBytes
	reconcileData.ResolvedHashes.Config = configHash

	_, tunnelHash, err := clabernetesutil.HashObjectYAML(
		reconcileData.ResolvedConfigs,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedHashes.Tunnels = tunnelHash

	for nodeName, nodeFilesFromURL := range owningTopology.Spec.Deployment.FilesFromURL {
		var nodeFilesFromURLHash string

		_, nodeFilesFromURLHash, err = clabernetesutil.HashObject(nodeFilesFromURL)
		if err != nil {
			return err
		}

		reconcileData.ResolvedHashes.FilesFromURL[nodeName] = nodeFilesFromURLHash

		if reconcileData.PreviousHashes.FilesFromURL[nodeName] != nodeFilesFromURLHash {
			// files from url hash has changed, need to smack the node so the configmap update
			// gets realized
			reconcileData.NodesNeedingReboot.Add(nodeName)
		}
	}

	imagePullSecretsBytes, imagePullSecretsHash, err := clabernetesutil.HashObjectYAML(
		owningTopology.Spec.ImagePull.PullSecrets,
	)
	if err != nil {
		return err
	}

	reconcileData.ResolvedHashes.ImagePullSecrets = imagePullSecretsHash

	if !reconcileData.ConfigMapHasChanges() {
		// the configs hashes match, nothing to do, should reconcile is false, and no error, *but*
		// because the services may force us to update the cr we are reconciling, and we haven't
		// processed the tunnel ids yet (because its slow and we are lazy), we need to copy the
		// *previous* tunnel data into our "current" tunnel data so we make sure to not update the
		// cr status with tunnel data with all zero for the tunnel ids
		reconcileData.ResolvedTunnels = reconcileData.PreviousTunnels

		return nil
	}

	AllocateTunnelIDs(
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
		owningTopology,
		reconcileData.ResolvedConfigs,
		reconcileData.ResolvedTunnels,
		owningTopology.Spec.Deployment.FilesFromURL,
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

// ReconcileServices reconciles all the services for a clabernetes Topology.
func (r *Reconciler) ReconcileServices(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
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
	owningTopology *clabernetesapisv1alpha1.Topology,
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
	owningTopology *clabernetesapisv1alpha1.Topology,
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
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	serviceTypeName := fmt.Sprintf("expose %s", clabernetesconstants.KubernetesService)

	if owningTopology.Status.ExposedPorts == nil {
		owningTopology.Status.ExposedPorts = map[string]*clabernetesapisv1alpha1.ExposedPorts{}

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
				reconcileData.ResolvedExposedPorts[existingCurrentServiceNodeName].LoadBalancerAddress = address //nolint:lll
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
		owningTopology.Status.ExposedPorts,
	)
	if err != nil {
		return err
	}

	if owningTopology.Status.ReconcileHashes.ExposedPorts != newNodeExposedPortsHash {
		reconcileData.ResolvedHashes.ExposedPorts = newNodeExposedPortsHash

		// our exposed hash stuff changed, we need to update the cr status
		reconcileData.ShouldUpdateResource = true
	}

	return nil
}

// ReconcilePersistentVolumeClaim reconciles the persistent volume claims used for persisting the
// containerlab working directory on nodes in a topology.
func (r *Reconciler) ReconcilePersistentVolumeClaim(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	pvcs, err := reconcileResolve(
		ctx,
		r,
		&k8scorev1.PersistentVolumeClaim{},
		&k8scorev1.PersistentVolumeClaimList{},
		clabernetesconstants.KubernetesPVC,
		owningTopology,
		reconcileData.ResolvedConfigs,
		r.persistentVolumeClaimReconciler.Resolve,
	)
	if err != nil {
		return err
	}

	r.Log.Info("pruning extraneous pvcs")

	for _, extraPVC := range pvcs.Extra {
		err = r.deleteObj(ctx, extraPVC, clabernetesconstants.KubernetesPVC)
		if err != nil {
			return err
		}
	}

	r.Log.Info("creating missing pvcs")

	renderedMissingPVCs := r.persistentVolumeClaimReconciler.RenderAll(
		owningTopology,
		pvcs.Missing,
	)

	for _, renderedMissingPVC := range renderedMissingPVCs {
		err = r.createObj(
			ctx,
			owningTopology,
			renderedMissingPVC,
			clabernetesconstants.KubernetesPVC,
		)
		if err != nil {
			return err
		}
	}

	r.Log.Info("enforcing desired state on existing deployments")

	for existingCurrentPVCNodeName, existingCurrentPVC := range pvcs.Current {
		renderedCurrentPVC := r.persistentVolumeClaimReconciler.Render(
			owningTopology,
			existingCurrentPVCNodeName,
			existingCurrentPVC,
		)

		err = ctrlruntimeutil.SetOwnerReference(
			owningTopology,
			renderedCurrentPVC,
			r.Client.Scheme(),
		)
		if err != nil {
			return err
		}

		if !r.persistentVolumeClaimReconciler.Conforms(
			existingCurrentPVC,
			renderedCurrentPVC,
			owningTopology.GetUID(),
		) {
			// only diff'ing spec since we *probably* only care about that part (minus metadata)
			r.diffIfDebug(existingCurrentPVC.Spec, renderedCurrentPVC.Spec)

			err = r.updateObj(
				ctx,
				renderedCurrentPVC,
				clabernetesconstants.KubernetesPVC,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) reconcileDeploymentsHandleRestarts(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
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

	var restartNodeError error

	for _, nodeName := range reconcileData.NodesNeedingReboot.Items() {
		if slices.Contains(deployments.Missing, nodeName) {
			// is a new node, don't restart, we'll deploy it soon
			continue
		}

		r.Log.Infof(
			"restarting the node '%s' as configurations have changed",
			nodeName,
		)

		r.diffIfDebug(
			reconcileData.PreviousConfigs[nodeName],
			reconcileData.ResolvedConfigs[nodeName],
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

				continue
			}

			r.Log.Warnf("failed fetching deployment for node %q, err: %s", nodeName, err)

			if restartNodeError == nil {
				restartNodeError = fmt.Errorf(
					"%w: encountered issue during node reboot process",
					claberneteserrors.ErrReconcile,
				)
			}

			continue
		}

		if nodeDeployment.Spec.Template.ObjectMeta.Annotations == nil {
			nodeDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}

		now := time.Now().Format(time.RFC3339)

		nodeDeployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = now //nolint:lll

		err = r.updateObj(ctx, nodeDeployment, clabernetesconstants.KubernetesDeployment)
		if err != nil {
			r.Log.Warnf("failed restarting deployment for node %q, err: %s", nodeName, err)

			if restartNodeError == nil {
				restartNodeError = fmt.Errorf(
					"%w: encountered issue during node reboot process",
					claberneteserrors.ErrReconcile,
				)
			}

			continue
		}
	}

	return restartNodeError
}

// ReconcileDeployments reconciles the deployments that make up a clabernetes Topology.
func (r *Reconciler) ReconcileDeployments(
	ctx context.Context,
	owningTopology *clabernetesapisv1alpha1.Topology,
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
			// only diff'ing spec since we *probably* only care about that part (minus metadata)
			r.diffIfDebug(existingCurrentDeployment.Spec, renderedCurrentDeployment.Spec)

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

func (r *Reconciler) diffIfDebug(a, b any) {
	if r.Log.GetLevel() != clabernetesconstants.Debug {
		return
	}

	diff, err := clabernetesutil.UnifiedDiff(
		a,
		b,
	)
	if err != nil {
		r.Log.Warnf(
			"failed generating diff. this only happened because logging"+
				" is at debug level, ignoring the error. err: %s",
			err,
		)
	} else {
		r.Log.Debugf("object diff: %s", diff)
	}
}

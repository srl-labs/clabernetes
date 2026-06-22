package node

import (
	"context"
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetestopology "github.com/srl-labs/clabernetes/controllers/topology"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconciler reconciles the per-node resources (ConfigMap, Deployment, fabric/expose Services, PVC)
// for a single Node custom resource. It deliberately *reuses* the existing Topology sub-reconcilers'
// render logic so the produced objects are byte-for-byte identical to what the monolithic Topology
// path produces today -- the only difference is that here the work is scoped to a single node and
// the rendered ConfigMap is per-node (rather than one ever-growing ConfigMap for the whole topology).
//
// NOTE (Phase 1): the renderers fundamentally take the owning *Topology (for node-invariant knobs
// like launcher image, expose flavor, scheduling, etc.), so this reconciler fetches it. The per-node
// state that actually scales -- the sub-topology definition and files -- comes from the Node itself.
// A later phase will make the Node fully self-contained so this Topology fetch can be dropped; see
// docs/design/0001-scale-node-link-crds.md.
type Reconciler struct {
	Log    claberneteslogging.Instance
	Client ctrlruntimeclient.Client

	configManagerGetter clabernetesconfig.ManagerGetterFunc

	configMapReconciler     *clabernetestopology.ConfigMapReconciler
	deploymentReconciler    *clabernetestopology.DeploymentReconciler
	serviceFabricReconciler *clabernetestopology.ServiceFabricReconciler
	serviceExposeReconciler *clabernetestopology.ServiceExposeReconciler
	pvcReconciler           *clabernetestopology.PersistentVolumeClaimReconciler
}

// NewReconciler returns a new Node Reconciler.
func NewReconciler(
	log claberneteslogging.Instance,
	client ctrlruntimeclient.Client,
	managerAppName,
	managerNamespace,
	criKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *Reconciler {
	return &Reconciler{
		Log:                 log,
		Client:              client,
		configManagerGetter: configManagerGetter,
		configMapReconciler: clabernetestopology.NewConfigMapReconciler(
			log,
			configManagerGetter,
		),
		deploymentReconciler: clabernetestopology.NewDeploymentReconciler(
			log,
			managerAppName,
			managerNamespace,
			criKind,
			configManagerGetter,
		),
		serviceFabricReconciler: clabernetestopology.NewServiceFabricReconciler(
			log,
			configManagerGetter,
		),
		serviceExposeReconciler: clabernetestopology.NewServiceExposeReconciler(
			log,
			configManagerGetter,
		),
		pvcReconciler: clabernetestopology.NewPersistentVolumeClaimReconciler(
			log,
			configManagerGetter,
		),
	}
}

// ReconcileNode reconciles all the resources that make up a single Node.
func (r *Reconciler) ReconcileNode(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
) error {
	owningTopology, err := r.getOwningTopology(ctx, node)
	if err != nil {
		return err
	}

	nodeName := node.Spec.NodeName

	// parse this node's single-node sub-topology out of its own spec -- this is the (bounded) state
	// that lets us avoid ever materializing a whole-topology object.
	nodeConfig := &clabernetesutilcontainerlab.Config{}

	err = yaml.Unmarshal([]byte(node.Spec.Definition), nodeConfig)
	if err != nil {
		return fmt.Errorf("failed unmarshalling node definition: %w", err)
	}

	singleNodeConfigs := map[string]*clabernetesutilcontainerlab.Config{
		nodeName: nodeConfig,
	}

	// build a minimal ReconcileData -- the expose reconciler derives a node's ports straight from
	// its resolved sub-topology config, so this single-node view is all it needs.
	reconcileData := &clabernetestopology.ReconcileData{
		ResolvedConfigs:      singleNodeConfigs,
		ResolvedExposedPorts: map[string]*clabernetesapisv1alpha1.ExposedPorts{},
	}

	configMapName, err := r.reconcileConfigMap(ctx, node, owningTopology, singleNodeConfigs)
	if err != nil {
		return err
	}

	err = r.reconcileDeployment(
		ctx,
		node,
		owningTopology,
		singleNodeConfigs,
		nodeName,
		configMapName,
	)
	if err != nil {
		return err
	}

	err = r.reconcileFabricService(ctx, node, owningTopology, nodeName)
	if err != nil {
		return err
	}

	err = r.reconcileExposeService(ctx, node, owningTopology, reconcileData, nodeName)
	if err != nil {
		return err
	}

	err = r.reconcilePersistentVolumeClaim(ctx, node, owningTopology, nodeName)
	if err != nil {
		return err
	}

	return r.reconcileStatus(ctx, node, owningTopology, nodeName)
}

// getOwningTopology fetches the Topology that owns this Node.
func (r *Reconciler) getOwningTopology(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
) (*clabernetesapisv1alpha1.Topology, error) {
	owningTopology := &clabernetesapisv1alpha1.Topology{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: node.GetNamespace(),
			Name:      node.Spec.TopologyName,
		},
		owningTopology,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed getting owning topology '%s/%s': %w",
			node.GetNamespace(),
			node.Spec.TopologyName,
			err,
		)
	}

	return owningTopology, nil
}

// perNodeConfigMapName returns the name of the per-node ConfigMap -- unlike the legacy single
// per-topology ConfigMap, each node gets its own so no single object grows with topology size.
func perNodeConfigMapName(node *clabernetesapisv1alpha1.Node) string {
	return fmt.Sprintf("%s-config", node.GetName())
}

// reconcileConfigMap renders and reconciles this node's own ConfigMap and returns its name.
func (r *Reconciler) reconcileConfigMap(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	singleNodeConfigs map[string]*clabernetesutilcontainerlab.Config,
) (string, error) {
	nodeName := node.Spec.NodeName

	var filesFromURL map[string][]clabernetesapisv1alpha1.FileFromURL
	if node.Spec.FilesFromURL != nil {
		filesFromURL = map[string][]clabernetesapisv1alpha1.FileFromURL{
			nodeName: node.Spec.FilesFromURL,
		}
	}

	imagePullSecretsBytes, _, err := clabernetesutil.HashObjectYAML(
		owningTopology.Spec.ImagePull.PullSecrets,
	)
	if err != nil {
		return "", err
	}

	rendered, err := r.configMapReconciler.Render(
		owningTopology,
		singleNodeConfigs,
		filesFromURL,
		string(imagePullSecretsBytes),
	)
	if err != nil {
		return "", err
	}

	// scope the ConfigMap to this node (the topology-wide render names it after the topology).
	configMapName := perNodeConfigMapName(node)
	rendered.Name = configMapName

	err = ctrlruntimeutil.SetOwnerReference(node, rendered, r.Client.Scheme())
	if err != nil {
		return "", err
	}

	existing := &k8scorev1.ConfigMap{}

	err = r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{Namespace: rendered.GetNamespace(), Name: configMapName},
		existing,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return configMapName, r.Client.Create(ctx, rendered)
		}

		return "", err
	}

	if r.configMapReconciler.Conforms(existing, rendered, node.GetUID()) {
		return configMapName, nil
	}

	rendered.ResourceVersion = existing.ResourceVersion

	return configMapName, r.Client.Update(ctx, rendered)
}

// reconcileDeployment renders and reconciles this node's Deployment, rewiring its config volume to
// the per-node ConfigMap.
func (r *Reconciler) reconcileDeployment(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	singleNodeConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName,
	configMapName string,
) error {
	render := func() (*k8sappsv1.Deployment, error) {
		deployment := r.deploymentReconciler.Render(owningTopology, singleNodeConfigs, nodeName)

		rewireConfigVolume(deployment, owningTopology.GetName(), configMapName)

		// point this node's launcher at its own per-node Connectivity object (instead of the
		// topology-wide one) so connectivity scales with the topology; see
		// docs/design/0001-scale-node-link-crds.md.
		rewireConnectivityEnv(
			deployment,
			clabernetestopology.PerNodeConnectivityName(owningTopology.GetName(), nodeName),
		)

		err := ctrlruntimeutil.SetOwnerReference(node, deployment, r.Client.Scheme())
		if err != nil {
			return nil, err
		}

		return deployment, nil
	}

	rendered, err := render()
	if err != nil {
		return err
	}

	existing := &k8sappsv1.Deployment{}

	err = r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: rendered.GetNamespace(),
			Name:      rendered.GetName(),
		},
		existing,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.Client.Create(ctx, rendered)
		}

		return err
	}

	if r.deploymentReconciler.Conforms(existing, rendered, node.GetUID()) {
		return nil
	}

	rendered.ResourceVersion = existing.ResourceVersion

	return r.Client.Update(ctx, rendered)
}

// rewireConfigVolume points the rendered deployment's clabernetes config volume at the per-node
// ConfigMap. The topology-wide deployment renderer references a single ConfigMap named after the
// Topology; in the decomposed path each node mounts its own.
func rewireConfigVolume(
	deployment *k8sappsv1.Deployment,
	owningTopologyName,
	configMapName string,
) {
	for i := range deployment.Spec.Template.Spec.Volumes {
		volume := &deployment.Spec.Template.Spec.Volumes[i]

		if volume.ConfigMap == nil {
			continue
		}

		if volume.ConfigMap.LocalObjectReference.Name != owningTopologyName {
			// some other (user) configmap mount, e.g. filesFromConfigMap -- leave it be.
			continue
		}

		volume.ConfigMap.LocalObjectReference.Name = configMapName
	}
}

// rewireConnectivityEnv sets the LauncherConnectivityNameEnv on every container of the rendered
// deployment so the launcher reads its per-node Connectivity object rather than the topology-wide
// one. The topology-wide deployment renderer does not set this env, so the launcher would otherwise
// fall back to the legacy object.
func rewireConnectivityEnv(deployment *k8sappsv1.Deployment, connectivityName string) {
	containers := deployment.Spec.Template.Spec.Containers

	for i := range containers {
		envs := containers[i].Env

		set := false

		for j := range envs {
			if envs[j].Name == clabernetesconstants.LauncherConnectivityNameEnv {
				envs[j].Value = connectivityName
				set = true

				break
			}
		}

		if !set {
			containers[i].Env = append(
				envs,
				k8scorev1.EnvVar{
					Name:  clabernetesconstants.LauncherConnectivityNameEnv,
					Value: connectivityName,
				},
			)
		}
	}
}

// reconcileFabricService renders and reconciles this node's "fabric" (inter-node) Service.
func (r *Reconciler) reconcileFabricService(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeName string,
) error {
	rendered := r.serviceFabricReconciler.Render(owningTopology, nodeName)

	err := ctrlruntimeutil.SetOwnerReference(node, rendered, r.Client.Scheme())
	if err != nil {
		return err
	}

	return r.createOrUpdateService(ctx, rendered, node.GetUID())
}

// reconcileExposeService renders and reconciles this node's "expose" Service (if exposing is
// enabled for the topology).
func (r *Reconciler) reconcileExposeService(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *clabernetestopology.ReconcileData,
	nodeName string,
) error {
	rendered := r.serviceExposeReconciler.Render(owningTopology, reconcileData, nodeName)
	if rendered == nil {
		// exposing is disabled for this topology, nothing to do.
		return nil
	}

	err := ctrlruntimeutil.SetOwnerReference(node, rendered, r.Client.Scheme())
	if err != nil {
		return err
	}

	return r.createOrUpdateService(ctx, rendered, node.GetUID())
}

// createOrUpdateService creates or updates a rendered Service, using the shared Service conformance
// check.
func (r *Reconciler) createOrUpdateService(
	ctx context.Context,
	rendered *k8scorev1.Service,
	expectedOwnerUID apimachinerytypes.UID,
) error {
	existing := &k8scorev1.Service{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: rendered.GetNamespace(),
			Name:      rendered.GetName(),
		},
		existing,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.Client.Create(ctx, rendered)
		}

		return err
	}

	if clabernetestopology.ServiceConforms(existing, rendered, expectedOwnerUID) {
		return nil
	}

	rendered.ResourceVersion = existing.ResourceVersion
	// preserve the cluster-assigned ClusterIP so updates don't get rejected.
	rendered.Spec.ClusterIP = existing.Spec.ClusterIP

	return r.Client.Update(ctx, rendered)
}

// reconcilePersistentVolumeClaim renders and reconciles this node's PVC when persistence is enabled.
func (r *Reconciler) reconcilePersistentVolumeClaim(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeName string,
) error {
	if !owningTopology.Spec.Deployment.Persistence.Enabled {
		return nil
	}

	existing := &k8scorev1.PersistentVolumeClaim{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: owningTopology.GetNamespace(),
			Name:      fmt.Sprintf("%s-%s", owningTopology.GetName(), nodeName),
		},
		existing,
	)

	notFound := apimachineryerrors.IsNotFound(err)
	if err != nil && !notFound {
		return err
	}

	var existingForRender *k8scorev1.PersistentVolumeClaim
	if !notFound {
		existingForRender = existing
	}

	rendered := r.pvcReconciler.Render(owningTopology, nodeName, existingForRender)

	err = ctrlruntimeutil.SetOwnerReference(node, rendered, r.Client.Scheme())
	if err != nil {
		return err
	}

	if notFound {
		return r.Client.Create(ctx, rendered)
	}

	if r.pvcReconciler.Conforms(existing, rendered, node.GetUID()) {
		return nil
	}

	rendered.ResourceVersion = existing.ResourceVersion

	return r.Client.Update(ctx, rendered)
}

// reconcileStatus reflects the node's Deployment availability into the Node's status.
func (r *Reconciler) reconcileStatus(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeName string,
) error {
	deploymentName := fmt.Sprintf("%s-%s", owningTopology.GetName(), nodeName)
	if clabernetestopology.ResolveTopologyRemovePrefix(owningTopology) {
		deploymentName = nodeName
	}

	deployment := &k8sappsv1.Deployment{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{Namespace: node.GetNamespace(), Name: deploymentName},
		deployment,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.setReady(ctx, node, false)
		}

		return err
	}

	ready := false

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == k8sappsv1.DeploymentAvailable &&
			condition.Status == k8scorev1.ConditionTrue {
			ready = true

			break
		}
	}

	return r.setReady(ctx, node, ready)
}

// setReady updates the Node status (idempotently) with the given readiness.
func (r *Reconciler) setReady(
	ctx context.Context,
	node *clabernetesapisv1alpha1.Node,
	ready bool,
) error {
	readiness := clabernetesconstants.NodeStatusNotReady
	if ready {
		readiness = clabernetesconstants.NodeStatusReady
	}

	if node.Status.Ready == ready && node.Status.Readiness == readiness {
		// nothing changed, avoid a needless write.
		return nil
	}

	node.Status.Ready = ready
	node.Status.Readiness = readiness

	return r.Client.Status().Update(ctx, node)
}

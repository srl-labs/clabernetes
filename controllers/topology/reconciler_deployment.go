package topology

import (
	"fmt"
	"reflect"
	"strings"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeploymentReconciler returns an instance of DeploymentReconciler.
func NewDeploymentReconciler(
	resourceKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *DeploymentReconciler {
	return &DeploymentReconciler{
		resourceKind:        resourceKind,
		configManagerGetter: configManagerGetter,
	}
}

// DeploymentReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating deployments for a
// clabernetes topology resource.
type DeploymentReconciler struct {
	resourceKind        string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of deployments that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ResolvedDeployments object
// that contains the missing, extra, and current deployments for the topology.
func (r *DeploymentReconciler) Resolve(
	ownedDeployments *k8sappsv1.DeploymentList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) (*clabernetescontrollers.ResolvedDeployments, error) {
	deployments := &clabernetescontrollers.ResolvedDeployments{
		Current: map[string]*k8sappsv1.Deployment{},
	}

	for i := range ownedDeployments.Items {
		labels := ownedDeployments.Items[i].Labels

		if labels == nil {
			return nil, fmt.Errorf(
				"%w: labels are nil, but we expect to see topology owner label here",
				claberneteserrors.ErrInvalidData,
			)
		}

		nodeName, ok := labels[clabernetesconstants.LabelTopologyNode]
		if !ok || nodeName == "" {
			return nil, fmt.Errorf(
				"%w: topology node label is missing or empty",
				claberneteserrors.ErrInvalidData,
			)
		}

		deployments.Current[nodeName] = &ownedDeployments.Items[i]
	}

	allNodes := make([]string, len(clabernetesConfigs))

	var nodeIdx int

	for nodeName := range clabernetesConfigs {
		allNodes[nodeIdx] = nodeName

		nodeIdx++
	}

	deployments.Missing = clabernetesutil.StringSliceDifference(
		deployments.CurrentDeploymentNames(),
		allNodes,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		allNodes,
		deployments.CurrentDeploymentNames(),
	)

	deployments.Extra = make([]*k8sappsv1.Deployment, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		deployments.Extra[idx] = deployments.Current[endpoint]
	}

	return deployments, nil
}

func (r *DeploymentReconciler) renderDeploymentBase(
	name,
	namespace,
	owningTopologyName,
	nodeName string,
) *k8sappsv1.Deployment {
	annotations, globalLabels := r.configManagerGetter().GetAllMetadata()

	deploymentName := fmt.Sprintf("%s-%s", owningTopologyName, nodeName)

	selectorLabels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: owningTopologyName,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	labels := make(map[string]string)

	for k, v := range selectorLabels {
		labels[k] = v
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	return &k8sappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: k8sappsv1.DeploymentSpec{
			Replicas:             clabernetesutil.ToPointer(int32(1)),
			RevisionHistoryLimit: clabernetesutil.ToPointer(int32(0)),
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: k8scorev1.PodSpec{
					Containers:         []k8scorev1.Container{},
					RestartPolicy:      "Always",
					ServiceAccountName: "default",
					Volumes:            []k8scorev1.Volume{},
				},
			},
		},
	}
}

func (r *DeploymentReconciler) renderDeploymentVolumes(
	deployment *k8sappsv1.Deployment,
	nodeName,
	configVolumeName,
	owningTopologyName string,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
) []k8scorev1.VolumeMount {
	volumes := []k8scorev1.Volume{
		{
			Name: configVolumeName,
			VolumeSource: k8scorev1.VolumeSource{
				ConfigMap: &k8scorev1.ConfigMapVolumeSource{
					LocalObjectReference: k8scorev1.LocalObjectReference{
						Name: owningTopologyName,
					},
				},
			},
		},
	}

	volumesFromConfigMaps := make([]clabernetesapistopologyv1alpha1.FileFromConfigMap, 0)

	volumeMountsFromCommonSpec := make([]k8scorev1.VolumeMount, 0)

	for _, fileFromConfigMap := range owningTopologyCommonSpec.FilesFromConfigMap {
		if fileFromConfigMap.NodeName != nodeName {
			continue
		}

		volumesFromConfigMaps = append(volumesFromConfigMaps, fileFromConfigMap)
	}

	for _, podVolume := range volumesFromConfigMaps {
		if !clabernetescontrollers.VolumeAlreadyMounted(
			podVolume.ConfigMapName,
			deployment.Spec.Template.Spec.Volumes,
		) {
			deployment.Spec.Template.Spec.Volumes = append(
				deployment.Spec.Template.Spec.Volumes,
				k8scorev1.Volume{
					Name: podVolume.ConfigMapName,
					VolumeSource: k8scorev1.VolumeSource{
						ConfigMap: &k8scorev1.ConfigMapVolumeSource{
							LocalObjectReference: k8scorev1.LocalObjectReference{
								Name: podVolume.ConfigMapName,
							},
						},
					},
				},
			)
		}

		volumeMount := k8scorev1.VolumeMount{
			Name:      podVolume.ConfigMapName,
			ReadOnly:  false,
			MountPath: fmt.Sprintf("/clabernetes/%s", podVolume.FilePath),
			SubPath:   podVolume.ConfigMapPath,
		}

		volumeMountsFromCommonSpec = append(
			volumeMountsFromCommonSpec,
			volumeMount,
		)
	}

	deployment.Spec.Template.Spec.Volumes = volumes

	return volumeMountsFromCommonSpec
}

func (r *DeploymentReconciler) renderDeploymentContainer(
	deployment *k8sappsv1.Deployment,
	nodeName,
	configVolumeName string,
	volumeMountsFromCommonSpec []k8scorev1.VolumeMount,
) {
	container := k8scorev1.Container{
		Name:       nodeName,
		WorkingDir: "/clabernetes",
		Image: clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.LauncherImageEnv,
			clabernetesconstants.LauncherDefaultImage,
		),
		Command: []string{"/clabernetes/manager", "launch"},
		Ports: []k8scorev1.ContainerPort{
			{
				Name:          "vxlan",
				ContainerPort: clabernetesconstants.VXLANServicePort,
				Protocol:      "UDP",
			},
		},
		VolumeMounts: []k8scorev1.VolumeMount{
			{
				Name:      configVolumeName,
				ReadOnly:  true,
				MountPath: "/clabernetes/topo.clab.yaml",
				SubPath:   nodeName,
			},
			{
				Name:      configVolumeName,
				ReadOnly:  true,
				MountPath: "/clabernetes/tunnels.yaml",
				SubPath:   fmt.Sprintf("%s-tunnels", nodeName),
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
		ImagePullPolicy: k8scorev1.PullPolicy(
			clabernetesutil.GetEnvStrOrDefault(
				clabernetesconstants.LauncherPullPolicyEnv,
				"IfNotPresent",
			),
		),
		SecurityContext: &k8scorev1.SecurityContext{
			// obviously we need privileged for dind setup
			Privileged: clabernetesutil.ToPointer(true),
			RunAsUser:  clabernetesutil.ToPointer(int64(0)),
		},
	}

	container.VolumeMounts = append(container.VolumeMounts, volumeMountsFromCommonSpec...)

	deployment.Spec.Template.Spec.Containers = []k8scorev1.Container{container}
}

func (r *DeploymentReconciler) renderDeploymentContainerEnv(
	deployment *k8sappsv1.Deployment,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
) {
	launcherLogLevel := clabernetesutil.GetEnvStrOrDefault(
		clabernetesconstants.LauncherLoggerLevelEnv,
		clabernetesconstants.Info,
	)

	if owningTopologyCommonSpec.LauncherLogLevel != "" {
		launcherLogLevel = owningTopologyCommonSpec.LauncherLogLevel
	}

	envs := []k8scorev1.EnvVar{
		{
			Name:  clabernetesconstants.LauncherLoggerLevelEnv,
			Value: launcherLogLevel,
		},
	}

	if owningTopologyCommonSpec.ContainerlabDebug {
		envs = append(
			envs,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherContainerlabDebug,
				Value: clabernetesconstants.True,
			},
		)
	}

	if len(owningTopologyCommonSpec.InsecureRegistries) > 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherInsecureRegistries,
				Value: strings.Join(owningTopologyCommonSpec.InsecureRegistries, ","),
			},
		)
	}

	deployment.Spec.Template.Spec.Containers[0].Env = envs
}

func (r *DeploymentReconciler) renderDeploymentContainerResources(
	deployment *k8sappsv1.Deployment,
	nodeName string,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) {
	nodeResources, nodeResourcesOk := owningTopologyCommonSpec.Resources[nodeName]
	if nodeResourcesOk {
		deployment.Spec.Template.Spec.Containers[0].Resources = nodeResources

		return
	}

	defaultResources, defaultResourcesOk := owningTopologyCommonSpec.Resources[clabernetesconstants.Default] //nolint:lll
	if defaultResourcesOk {
		deployment.Spec.Template.Spec.Containers[0].Resources = defaultResources

		return
	}

	resources := r.configManagerGetter().GetResourcesForContainerlabKind(
		clabernetesConfigs[nodeName].Topology.GetNodeKindType(nodeName),
	)

	if resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *resources
	}
}

// Render accepts an object (just for name/namespace reasons) a mapping of clabernetes
// sub-topology configs and a node name and renders the final deployment for this node.
func (r *DeploymentReconciler) Render(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) *k8sappsv1.Deployment {
	owningTopologyName := owningTopology.GetName()

	owningTopologyCommonSpec := owningTopology.GetTopologyCommonSpec()

	configVolumeName := fmt.Sprintf("%s-config", owningTopologyName)

	deployment := r.renderDeploymentBase(
		fmt.Sprintf("%s-%s", owningTopologyName, nodeName),
		owningTopology.GetNamespace(),
		owningTopologyName,
		nodeName,
	)

	volumeMountsFromCommonSpec := r.renderDeploymentVolumes(
		deployment,
		nodeName,
		configVolumeName,
		owningTopologyName,
		&owningTopologyCommonSpec,
	)

	r.renderDeploymentContainer(
		deployment,
		nodeName,
		configVolumeName,
		volumeMountsFromCommonSpec,
	)

	r.renderDeploymentContainerEnv(
		deployment,
		&owningTopologyCommonSpec,
	)

	r.renderDeploymentContainerResources(
		deployment,
		nodeName,
		&owningTopologyCommonSpec,
		clabernetesConfigs,
	)

	return deployment
}

// RenderAll accepts an object (just for name/namespace reasons) a mapping of clabernetes
// sub-topology configs and a list of node names and renders the final deployment for the given
// nodes.
func (r *DeploymentReconciler) RenderAll(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeNames []string,
) []*k8sappsv1.Deployment {
	deployments := make([]*k8sappsv1.Deployment, len(nodeNames))

	for idx, nodeName := range nodeNames {
		deployments[idx] = r.Render(
			owningTopology,
			clabernetesConfigs,
			nodeName,
		)
	}

	return deployments
}

// Conforms checks if the existingDeployment conforms with the renderedDeployment.
func (r *DeploymentReconciler) Conforms(
	existingDeployment,
	renderedDeployment *k8sappsv1.Deployment,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	if !reflect.DeepEqual(existingDeployment.Spec.Replicas, renderedDeployment.Spec.Replicas) {
		return false
	}

	if !reflect.DeepEqual(existingDeployment.Spec.Selector, renderedDeployment.Spec.Selector) {
		return false
	}

	if !clabernetescontrollers.ContainersEqual(
		existingDeployment.Spec.Template.Spec.Containers,
		renderedDeployment.Spec.Template.Spec.Containers,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.ServiceAccountName,
		renderedDeployment.Spec.Template.Spec.ServiceAccountName,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.RestartPolicy,
		renderedDeployment.Spec.Template.Spec.RestartPolicy,
	) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
		existingDeployment.ObjectMeta.Annotations,
		renderedDeployment.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
		existingDeployment.ObjectMeta.Labels,
		renderedDeployment.ObjectMeta.Labels,
	) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
		existingDeployment.Spec.Template.ObjectMeta.Annotations,
		renderedDeployment.Spec.Template.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
		existingDeployment.Spec.Template.ObjectMeta.Labels,
		renderedDeployment.Spec.Template.ObjectMeta.Labels,
	) {
		return false
	}

	if len(existingDeployment.ObjectMeta.OwnerReferences) != 1 {
		// we should have only one owner reference, the extractor
		return false
	}

	if existingDeployment.ObjectMeta.OwnerReferences[0].UID != expectedOwnerUID {
		// owner ref uid is not us
		return false
	}

	return true
}

// DetermineNodesNeedingRestart accepts a mapping of the previously stored clabernetes
// sub-topologies and the current reconcile loops rendered topologies and returns a slice of node
// names whose deployments need restarting due to configuration changes.
func (r *DeploymentReconciler) DetermineNodesNeedingRestart(
	previousClabernetesConfigs,
	currentClabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) []string {
	var nodesNeedingRestart []string

	for nodeName, nodeConfig := range currentClabernetesConfigs {
		_, nodeExistedBefore := previousClabernetesConfigs[nodeName]
		if !nodeExistedBefore {
			continue
		}

		if !reflect.DeepEqual(nodeConfig, previousClabernetesConfigs[nodeName]) {
			nodesNeedingRestart = append(
				nodesNeedingRestart,
				nodeName,
			)
		}
	}

	return nodesNeedingRestart
}

package reconciler

import (
	"fmt"
	"reflect"
	"strings"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"

	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeploymentReconciler returns an instance of DeploymentReconciler.
func NewDeploymentReconciler(
	log claberneteslogging.Instance,
	managerAppName,
	managerNamespace,
	owningTopologyKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
	criKind,
	imagePullThroughMode string,
) *DeploymentReconciler {
	return &DeploymentReconciler{
		log:                  log,
		managerAppName:       managerAppName,
		managerNamespace:     managerNamespace,
		owningTopologyKind:   owningTopologyKind,
		configManagerGetter:  configManagerGetter,
		criKind:              criKind,
		imagePullThroughMode: imagePullThroughMode,
	}
}

// DeploymentReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating deployments for a
// clabernetes topology resource.
type DeploymentReconciler struct {
	log                  claberneteslogging.Instance
	managerAppName       string
	managerNamespace     string
	owningTopologyKind   string
	configManagerGetter  clabernetesconfig.ManagerGetterFunc
	criKind              string
	imagePullThroughMode string
}

// Resolve accepts a mapping of clabernetes configs and a list of deployments that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current deployments for the topology.
func (r *DeploymentReconciler) Resolve(
	ownedDeployments *k8sappsv1.DeploymentList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	_ clabernetesapistopologyv1alpha1.TopologyCommonObject,
) (*clabernetesutil.ObjectDiffer[*k8sappsv1.Deployment], error) {
	deployments := &clabernetesutil.ObjectDiffer[*k8sappsv1.Deployment]{
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

	deployments.SetMissing(allNodes)
	deployments.SetExtra(allNodes)

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

	labels := map[string]string{
		clabernetesconstants.LabelTopologyKind: r.owningTopologyKind,
	}

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
					Hostname:           nodeName,
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

	volumeMountsFromCommonSpec := make([]k8scorev1.VolumeMount, 0)

	// if we have containerd cri *and* pull through mode is auto or always, we need to mount the
	// containerd sock
	if r.imagePullThroughMode != clabernetesconstants.ImagePullThroughModeNever &&
		owningTopologyCommonSpec.ImagePullThroughOverride != clabernetesconstants.ImagePullThroughModeNever { //nolint:lll
		var path string

		var subPath string

		switch r.criKind {
		case clabernetesconstants.KubernetesCRIContainerd:
			path = clabernetesconstants.KubernetesCRISockContainerdPath

			subPath = clabernetesconstants.KubernetesCRISockContainerd
		default:
			r.log.Warnf(
				"image pull through mode is auto or always but cri kind is not containerd!"+
					" got cri kind %q",
				r.criKind,
			)
		}

		if path != "" && subPath != "" {
			volumes = append(
				volumes,
				k8scorev1.Volume{
					Name: "cri-sock",
					VolumeSource: k8scorev1.VolumeSource{
						HostPath: &k8scorev1.HostPathVolumeSource{
							Path: path,
						},
					},
				},
			)

			volumeMountsFromCommonSpec = append(
				volumeMountsFromCommonSpec,
				k8scorev1.VolumeMount{
					Name:     "cri-sock",
					ReadOnly: true,
					MountPath: fmt.Sprintf(
						"%s/%s",
						clabernetesconstants.LauncherCRISockPath,
						subPath,
					),
					SubPath: subPath,
				},
			)
		}
	}

	volumesFromConfigMaps := make([]clabernetesapistopologyv1alpha1.FileFromConfigMap, 0)

	volumesFromConfigMaps = append(
		volumesFromConfigMaps,
		owningTopologyCommonSpec.FilesFromConfigMap[nodeName]...,
	)

	for _, podVolume := range volumesFromConfigMaps {
		if !clabernetesutilkubernetes.VolumeAlreadyMounted(
			podVolume.ConfigMapName,
			volumes,
		) {
			volumes = append(
				volumes,
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
			{
				Name:      configVolumeName,
				ReadOnly:  true,
				MountPath: "/clabernetes/files-from-url.yaml",
				SubPath:   fmt.Sprintf("%s-files-from-url", nodeName),
			},
			{
				Name:      configVolumeName,
				ReadOnly:  true,
				MountPath: "/clabernetes/configured-pull-secrets.yaml",
				SubPath:   "configured-pull-secrets",
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
	}

	container.VolumeMounts = append(container.VolumeMounts, volumeMountsFromCommonSpec...)

	deployment.Spec.Template.Spec.Containers = []k8scorev1.Container{container}
}

func (r *DeploymentReconciler) renderDeploymentContainerEnv(
	deployment *k8sappsv1.Deployment,
	nodeName,
	owningTopologyName string,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) {
	launcherLogLevel := clabernetesutil.GetEnvStrOrDefault(
		clabernetesconstants.LauncherLoggerLevelEnv,
		clabernetesconstants.Info,
	)

	if owningTopologyCommonSpec.LauncherLogLevel != "" {
		launcherLogLevel = owningTopologyCommonSpec.LauncherLogLevel
	}

	imagePullThroughMode := r.imagePullThroughMode
	if owningTopologyCommonSpec.ImagePullThroughOverride != "" {
		imagePullThroughMode = owningTopologyCommonSpec.ImagePullThroughOverride
	}

	envs := []k8scorev1.EnvVar{
		{
			Name: clabernetesconstants.NodeNameEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: clabernetesconstants.PodNameEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: clabernetesconstants.PodNamespaceEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  clabernetesconstants.AppNameEnv,
			Value: r.managerAppName,
		},
		{
			Name:  clabernetesconstants.ManagerNamespaceEnv,
			Value: r.managerNamespace,
		},
		{
			Name:  clabernetesconstants.LauncherCRIKindEnv,
			Value: r.criKind,
		},
		{
			Name:  clabernetesconstants.LauncherImagePullThroughModeEnv,
			Value: imagePullThroughMode,
		},
		{
			Name:  clabernetesconstants.LauncherLoggerLevelEnv,
			Value: launcherLogLevel,
		},
		{
			Name:  clabernetesconstants.LauncherTopologyNameEnv,
			Value: owningTopologyName,
		},
		{
			Name:  clabernetesconstants.LauncherNodeNameEnv,
			Value: nodeName,
		},
		{
			Name:  clabernetesconstants.LauncherNodeImageEnv,
			Value: clabernetesConfigs[nodeName].Topology.GetNodeImage(nodeName),
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
		envs = append(
			envs,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherInsecureRegistries,
				Value: strings.Join(owningTopologyCommonSpec.InsecureRegistries, ","),
			},
		)
	}

	if owningTopologyCommonSpec.PrivilegedLauncher {
		envs = append(
			envs,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherPrivilegedEnv,
				Value: clabernetesconstants.True,
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

func (r *DeploymentReconciler) renderDeploymentContainerPrivileges(
	deployment *k8sappsv1.Deployment,
	nodeName string,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
) {
	if owningTopologyCommonSpec.PrivilegedLauncher {
		deployment.Spec.Template.Spec.Containers[0].SecurityContext = &k8scorev1.SecurityContext{
			Privileged: clabernetesutil.ToPointer(true),
			RunAsUser:  clabernetesutil.ToPointer(int64(0)),
		}

		return
	}

	// w/out this set you cant remount /sys/fs/cgroup, /proc, and /proc/sys; note that the part
	// after the "/" needs to be the name of the container this applies to -- in our case (for now?)
	// this will always just be the node name
	deployment.ObjectMeta.Annotations[fmt.Sprintf(
		"%s/%s", "container.apparmor.security.beta.kubernetes.io", nodeName,
	)] = "unconfined"

	deployment.Spec.Template.Spec.Containers[0].SecurityContext = &k8scorev1.SecurityContext{
		Privileged: clabernetesutil.ToPointer(false),
		RunAsUser:  clabernetesutil.ToPointer(int64(0)),
		Capabilities: &k8scorev1.Capabilities{
			Add: []k8scorev1.Capability{
				// docker says we need these ones:
				// https://github.com/moby/moby/blob/master/oci/caps/defaults.go#L6-L19
				"CHOWN",
				"DAC_OVERRIDE",
				"FSETID",
				"FOWNER",
				"MKNOD",
				"NET_RAW",
				"SETGID",
				"SETUID",
				"SETFCAP",
				"SETPCAP",
				"NET_BIND_SERVICE",
				"SYS_CHROOT",
				"KILL",
				"AUDIT_WRITE",
				// docker doesnt say we need this but surely we do otherwise cant connect to
				// daemon
				"NET_ADMIN",
				// cant untar/load image w/out this it seems
				// https://github.com/moby/moby/issues/43086
				"SYS_ADMIN",
				// this it seems we need otherwise we get some issues finding child pid of
				// containers and when we "docker run" it craps out
				"SYS_RESOURCE",
				// and some more that we needed to boot srl
				"LINUX_IMMUTABLE",
				"SYS_BOOT",
				"SYS_TIME",
				"SYS_MODULE",
				"SYS_RAWIO",
				"SYS_PTRACE",
				// and some more that we need to run xdp lc manager in srl, and probably others!?
				"SYS_NICE",
				"IPC_LOCK",
				// the rest for convenience of adding more later if needed
				// "MAC_OVERRIDE",
				// "MAC_ADMIN",
				// "BPF",
				// "PERFMON",
				// "NET_BROADCAST",
				// "DAC_READ_SEARCH",
				// "SYSLOG",
				// "WAKE_ALARM",
				// "BLOCK_SUSPEND",
				// "AUDIT_READ",
				// "LEASE",
				// "CHECKPOINT_RESTORE",
				// "SYS_TTY_CONFIG",
				// "SYS_PACCT",
				// "IPC_OWNER",
			},
		},
	}
}

func (r *DeploymentReconciler) renderDeploymentDevices(
	deployment *k8sappsv1.Deployment,
	owningTopologyCommonSpec *clabernetesapistopologyv1alpha1.TopologyCommonSpec,
) {
	if owningTopologyCommonSpec.PrivilegedLauncher {
		// launcher is privileged, no need to mount devices explicitly
		return
	}

	// add volumes for devices we care about
	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		[]k8scorev1.Volume{
			{
				Name: "dev-kvm",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: "/dev/kvm",
					},
				},
			},
			{
				Name: "dev-fuse",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: "/dev/fuse",
					},
				},
			},
			{
				Name: "dev-net-tun",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: "/dev/net/tun",
					},
				},
			},
		}...,
	)

	// then mount them in our container (launchers (for now?!) only ever have the one container)
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		[]k8scorev1.VolumeMount{
			{
				Name:      "dev-kvm",
				ReadOnly:  true,
				MountPath: "/dev/kvm",
			},
			{
				Name:      "dev-fuse",
				ReadOnly:  true,
				MountPath: "/dev/fuse",
			},
			{
				Name:      "dev-net-tun",
				ReadOnly:  true,
				MountPath: "/dev/net/tun",
			},
		}...,
	)
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the final deployment for this node.
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
		nodeName,
		owningTopologyName,
		&owningTopologyCommonSpec,
		clabernetesConfigs,
	)

	r.renderDeploymentContainerResources(
		deployment,
		nodeName,
		&owningTopologyCommonSpec,
		clabernetesConfigs,
	)

	r.renderDeploymentContainerPrivileges(
		deployment,
		nodeName,
		&owningTopologyCommonSpec,
	)

	r.renderDeploymentDevices(
		deployment,
		&owningTopologyCommonSpec,
	)

	return deployment
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the final deployments for the given nodes.
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

	if renderedDeployment.Spec.Template.Spec.Hostname !=
		existingDeployment.Spec.Template.Spec.Hostname {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.Volumes,
		renderedDeployment.Spec.Template.Spec.Volumes,
	) {
		return false
	}

	if !clabernetesutilkubernetes.ContainersEqual(
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

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingDeployment.ObjectMeta.Annotations,
		renderedDeployment.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingDeployment.ObjectMeta.Labels,
		renderedDeployment.ObjectMeta.Labels,
	) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingDeployment.Spec.Template.ObjectMeta.Annotations,
		renderedDeployment.Spec.Template.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
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

// DetermineNodesNeedingRestart accepts reconcile data (which contains the previous and current
// rendered sub-topologies) and updates the reconcile data NodesNeedingReboot set with each node
// that needs restarting due to configuration changes.
func (r *DeploymentReconciler) DetermineNodesNeedingRestart(
	reconcileData *ReconcileData,
) {
	for nodeName, nodeConfig := range reconcileData.ResolvedConfigs {
		_, nodeExistedBefore := reconcileData.PreviousConfigs[nodeName]
		if !nodeExistedBefore {
			continue
		}

		if !reflect.DeepEqual(nodeConfig, reconcileData.PreviousConfigs[nodeName]) {
			reconcileData.NodesNeedingReboot.Add(nodeName)
		}
	}
}

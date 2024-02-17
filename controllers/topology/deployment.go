package topology

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// NewDeploymentReconciler returns an instance of DeploymentReconciler.
func NewDeploymentReconciler(
	log claberneteslogging.Instance,
	managerAppName,
	managerNamespace,
	criKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *DeploymentReconciler {
	return &DeploymentReconciler{
		log:                 log,
		managerAppName:      managerAppName,
		managerNamespace:    managerNamespace,
		criKind:             criKind,
		configManagerGetter: configManagerGetter,
	}
}

// DeploymentReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating deployments for a
// clabernetes topology resource.
type DeploymentReconciler struct {
	log                 claberneteslogging.Instance
	managerAppName      string
	managerNamespace    string
	criKind             string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of deployments that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current deployments for the topology.
func (r *DeploymentReconciler) Resolve(
	ownedDeployments *k8sappsv1.DeploymentList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	_ *clabernetesapisv1alpha1.Topology,
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

	selectorLabels := map[string]string{
		clabernetesconstants.LabelKubernetesName: name,
		clabernetesconstants.LabelApp:            clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:           name,
		clabernetesconstants.LabelTopologyOwner:  owningTopologyName,
		clabernetesconstants.LabelTopologyNode:   nodeName,
	}

	labels := map[string]string{}

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
			Strategy: k8sappsv1.DeploymentStrategy{
				// in our case there is no (current?) need for more gracefully updating our
				// deployments, so just yolo recreate them instead...
				Type:          k8sappsv1.RecreateDeploymentStrategyType,
				RollingUpdate: nil,
			},
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
					ServiceAccountName: launcherServiceAccountName(),
					Volumes:            []k8scorev1.Volume{},
					Hostname:           nodeName,
				},
			},
		},
	}
}

func (r *DeploymentReconciler) renderDeploymentScheduling(
	deployment *k8sappsv1.Deployment,
	owningTopology *clabernetesapisv1alpha1.Topology,
) {
	nodeSelector := owningTopology.Spec.Deployment.Scheduling.NodeSelector
	tolerations := owningTopology.Spec.Deployment.Scheduling.Tolerations

	deployment.Spec.Template.Spec.NodeSelector = nodeSelector
	deployment.Spec.Template.Spec.Tolerations = tolerations
}

func (r *DeploymentReconciler) renderDeploymentVolumes(
	deployment *k8sappsv1.Deployment,
	nodeName,
	configVolumeName,
	owningTopologyName string,
	owningTopology *clabernetesapisv1alpha1.Topology,
) []k8scorev1.VolumeMount {
	volumes := []k8scorev1.Volume{
		{
			Name: configVolumeName,
			VolumeSource: k8scorev1.VolumeSource{
				ConfigMap: &k8scorev1.ConfigMapVolumeSource{
					LocalObjectReference: k8scorev1.LocalObjectReference{
						Name: owningTopologyName,
					},
					DefaultMode: clabernetesutil.ToPointer(
						int32(clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute),
					),
				},
			},
		},
	}

	volumeMountsFromCommonSpec := make([]k8scorev1.VolumeMount, 0)

	criPath, criSubPath := r.renderDeploymentVolumesGetCRISockPath(owningTopology)

	if criPath != "" && criSubPath != "" {
		volumes = append(
			volumes,
			k8scorev1.Volume{
				Name: "cri-sock",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: criPath,
						Type: clabernetesutil.ToPointer(k8scorev1.HostPathType("")),
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
					criSubPath,
				),
				SubPath: criSubPath,
			},
		)
	}

	dockerDaemonConfigSecret := owningTopology.Spec.ImagePull.DockerDaemonConfig
	if dockerDaemonConfigSecret == "" {
		dockerDaemonConfigSecret = r.configManagerGetter().GetDockerDaemonConfig()
	}

	if dockerDaemonConfigSecret != "" {
		volumes = append(
			volumes,
			k8scorev1.Volume{
				Name: "docker-daemon-config",
				VolumeSource: k8scorev1.VolumeSource{
					Secret: &k8scorev1.SecretVolumeSource{
						SecretName: dockerDaemonConfigSecret,
						DefaultMode: clabernetesutil.ToPointer(
							int32(clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute),
						),
					},
				},
			},
		)

		volumeMountsFromCommonSpec = append(
			volumeMountsFromCommonSpec,
			k8scorev1.VolumeMount{
				Name:      "docker-daemon-config",
				ReadOnly:  true,
				MountPath: "/etc/docker",
			},
		)
	}

	volumesFromConfigMaps := make([]clabernetesapisv1alpha1.FileFromConfigMap, 0)

	volumesFromConfigMaps = append(
		volumesFromConfigMaps,
		owningTopology.Spec.Deployment.FilesFromConfigMap[nodeName]...,
	)

	for _, podVolume := range volumesFromConfigMaps {
		volumeName := clabernetesutilkubernetes.EnforceDNSLabelConvention(
			clabernetesutilkubernetes.SafeConcatNameKubernetes(
				podVolume.ConfigMapName,
				podVolume.ConfigMapPath,
			),
		)

		var mode *int32

		switch podVolume.Mode {
		case clabernetesconstants.FileModeRead:
			mode = clabernetesutil.ToPointer(
				int32(clabernetesconstants.PermissionsEveryoneRead),
			)
		case clabernetesconstants.FileModeExecute:
			mode = clabernetesutil.ToPointer(
				int32(clabernetesconstants.PermissionsEveryoneReadExecute),
			)
		default:
			mode = nil
		}

		volumes = append(
			volumes,
			k8scorev1.Volume{
				Name: volumeName,
				VolumeSource: k8scorev1.VolumeSource{
					ConfigMap: &k8scorev1.ConfigMapVolumeSource{
						LocalObjectReference: k8scorev1.LocalObjectReference{
							Name: podVolume.ConfigMapName,
						},
						DefaultMode: mode,
					},
				},
			},
		)

		volumeMount := k8scorev1.VolumeMount{
			Name:      volumeName,
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

func (r *DeploymentReconciler) renderDeploymentVolumesGetCRISockPath(
	owningTopology *clabernetesapisv1alpha1.Topology,
) (path, subPath string) {
	if owningTopology.Spec.ImagePull.PullThroughOverride == clabernetesconstants.ImagePullThroughModeNever { //nolint:lll
		// obviously the topology is set to *never*, so nothing to do...
		return path, subPath
	}

	if owningTopology.Spec.ImagePull.PullThroughOverride == "" && r.configManagerGetter().
		GetImagePullThroughMode() == clabernetesconstants.ImagePullThroughModeNever {
		// our specific topology is setting is unset, so we default to the global value, if that
		// is never then we are obviously done here
		return path, subPath
	}

	criSockOverrideFullPath := r.configManagerGetter().GetImagePullCriSockOverride()
	if criSockOverrideFullPath != "" {
		path, subPath = filepath.Split(criSockOverrideFullPath)

		if path == "" {
			r.log.Warn(
				"image pull cri path override is set, but failed to parse path/subpath," +
					" will skip mounting cri sock",
			)

			return path, subPath
		}
	} else {
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
	}

	return path, subPath
}

func (r *DeploymentReconciler) renderDeploymentContainer(
	deployment *k8sappsv1.Deployment,
	nodeName,
	configVolumeName string,
	volumeMountsFromCommonSpec []k8scorev1.VolumeMount,
	owningTopology *clabernetesapisv1alpha1.Topology,
) {
	image := owningTopology.Spec.Deployment.LauncherImage
	if image == "" {
		image = r.configManagerGetter().GetLauncherImage()
	}

	imagePullPolicy := owningTopology.Spec.Deployment.LauncherImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = r.configManagerGetter().GetLauncherImagePullPolicy()
	}

	container := k8scorev1.Container{
		Name:       nodeName,
		WorkingDir: "/clabernetes",
		Image:      image,
		Command:    []string{"/clabernetes/manager", "launch"},
		Ports: []k8scorev1.ContainerPort{
			{
				Name:          clabernetesconstants.ConnectivityVXLAN,
				ContainerPort: clabernetesconstants.VXLANServicePort,
				Protocol:      clabernetesconstants.UDP,
			},
			{
				Name:          clabernetesconstants.ConnectivitySlurpeeth,
				ContainerPort: clabernetesconstants.SlurpeethServicePort,
				Protocol:      clabernetesconstants.TCP,
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
		ImagePullPolicy:          k8scorev1.PullPolicy(imagePullPolicy),
	}

	container.VolumeMounts = append(container.VolumeMounts, volumeMountsFromCommonSpec...)

	deployment.Spec.Template.Spec.Containers = []k8scorev1.Container{container}
}

func (r *DeploymentReconciler) renderDeploymentContainerEnv(
	deployment *k8sappsv1.Deployment,
	nodeName,
	owningTopologyName string,
	owningTopology *clabernetesapisv1alpha1.Topology,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) {
	launcherLogLevel := owningTopology.Spec.Deployment.LauncherLogLevel
	if launcherLogLevel == "" {
		launcherLogLevel = r.configManagerGetter().GetLauncherLogLevel()
	}

	imagePullThroughMode := owningTopology.Spec.ImagePull.PullThroughOverride
	if owningTopology.Spec.ImagePull.PullThroughOverride == "" {
		imagePullThroughMode = r.configManagerGetter().GetImagePullThroughMode()
	}

	criKind := r.configManagerGetter().GetImagePullCriKindOverride()
	if criKind == "" {
		criKind = r.criKind
	}

	nodeImage := clabernetesConfigs[nodeName].Topology.GetNodeImage(nodeName)
	if nodeImage == "" {
		r.log.Warnf(
			"could not parse image for node %q, topology in question printined in debug log",
			nodeName,
		)

		subTopologyBytes, err := json.MarshalIndent(clabernetesConfigs[nodeName], "", "    ")
		if err != nil {
			r.log.Warnf("failed marshaling topology, error: %s", err)
		} else {
			r.log.Debugf("node topology:\n%s", string(subTopologyBytes))
		}
	}

	envs := []k8scorev1.EnvVar{
		{
			Name: clabernetesconstants.NodeNameEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.nodeName",
				},
			},
		},
		{
			Name: clabernetesconstants.PodNameEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name: clabernetesconstants.PodNamespaceEnv,
			ValueFrom: &k8scorev1.EnvVarSource{
				FieldRef: &k8scorev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
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
			Value: criKind,
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
			Value: nodeImage,
		},
		{
			Name:  clabernetesconstants.LauncherConnectivityKind,
			Value: owningTopology.Spec.Connectivity,
		},
	}

	if ResolveGlobalVsTopologyBool(
		r.configManagerGetter().GetContainerlabDebug(),
		owningTopology.Spec.Deployment.ContainerlabDebug,
	) {
		envs = append(
			envs,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherContainerlabDebug,
				Value: clabernetesconstants.True,
			},
		)
	}

	if len(owningTopology.Spec.ImagePull.InsecureRegistries) > 0 {
		envs = append(
			envs,
			k8scorev1.EnvVar{
				Name:  clabernetesconstants.LauncherInsecureRegistries,
				Value: strings.Join(owningTopology.Spec.ImagePull.InsecureRegistries, ","),
			},
		)
	}

	if ResolveGlobalVsTopologyBool(
		r.configManagerGetter().GetPrivilegedLauncher(),
		owningTopology.Spec.Deployment.PrivilegedLauncher,
	) {
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
	owningTopology *clabernetesapisv1alpha1.Topology,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) {
	nodeResources, nodeResourcesOk := owningTopology.Spec.Deployment.Resources[nodeName]
	if nodeResourcesOk {
		deployment.Spec.Template.Spec.Containers[0].Resources = nodeResources

		return
	}

	defaultResources, defaultResourcesOk := owningTopology.Spec.Deployment.Resources[clabernetesconstants.Default] //nolint:lll
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
	owningTopology *clabernetesapisv1alpha1.Topology,
) {
	if ResolveGlobalVsTopologyBool(
		r.configManagerGetter().GetPrivilegedLauncher(),
		owningTopology.Spec.Deployment.PrivilegedLauncher,
	) {
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
	owningTopology *clabernetesapisv1alpha1.Topology,
) {
	if ResolveGlobalVsTopologyBool(
		r.configManagerGetter().GetPrivilegedLauncher(),
		owningTopology.Spec.Deployment.PrivilegedLauncher,
	) {
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
						Type: clabernetesutil.ToPointer(k8scorev1.HostPathType("")),
					},
				},
			},
			{
				Name: "dev-fuse",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: "/dev/fuse",
						Type: clabernetesutil.ToPointer(k8scorev1.HostPathType("")),
					},
				},
			},
			{
				Name: "dev-net-tun",
				VolumeSource: k8scorev1.VolumeSource{
					HostPath: &k8scorev1.HostPathVolumeSource{
						Path: "/dev/net/tun",
						Type: clabernetesutil.ToPointer(k8scorev1.HostPathType("")),
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

func (r *DeploymentReconciler) renderDeploymentPersistence(
	deployment *k8sappsv1.Deployment,
	nodeName,
	owningTopologyName string,
	owningTopology *clabernetesapisv1alpha1.Topology,
) {
	if !owningTopology.Spec.Deployment.Persistence.Enabled {
		return
	}

	volumeName := "containerlab-directory-persistence"

	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		k8scorev1.Volume{
			Name: volumeName,
			VolumeSource: k8scorev1.VolumeSource{
				PersistentVolumeClaim: &k8scorev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-%s", owningTopologyName, nodeName),
					ReadOnly:  false,
				},
			},
		},
	)

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		k8scorev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  false,
			MountPath: fmt.Sprintf("/clabernetes/clab-clabernetes-%s", nodeName),
		},
	)
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the final deployment for this node.
func (r *DeploymentReconciler) Render(
	owningTopology *clabernetesapisv1alpha1.Topology,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) *k8sappsv1.Deployment {
	owningTopologyName := owningTopology.GetName()

	deploymentName := fmt.Sprintf("%s-%s", owningTopologyName, nodeName)

	if ResolveTopologyRemovePrefix(owningTopology) {
		deploymentName = nodeName
	}

	configVolumeName := fmt.Sprintf("%s-config", owningTopologyName)

	deployment := r.renderDeploymentBase(
		deploymentName,
		owningTopology.GetNamespace(),
		owningTopologyName,
		nodeName,
	)

	r.renderDeploymentScheduling(
		deployment,
		owningTopology,
	)

	volumeMountsFromCommonSpec := r.renderDeploymentVolumes(
		deployment,
		nodeName,
		configVolumeName,
		owningTopologyName,
		owningTopology,
	)

	r.renderDeploymentContainer(
		deployment,
		nodeName,
		configVolumeName,
		volumeMountsFromCommonSpec,
		owningTopology,
	)

	r.renderDeploymentContainerEnv(
		deployment,
		nodeName,
		owningTopologyName,
		owningTopology,
		clabernetesConfigs,
	)

	r.renderDeploymentContainerResources(
		deployment,
		nodeName,
		owningTopology,
		clabernetesConfigs,
	)

	r.renderDeploymentContainerPrivileges(
		deployment,
		nodeName,
		owningTopology,
	)

	r.renderDeploymentDevices(
		deployment,
		owningTopology,
	)

	r.renderDeploymentPersistence(
		deployment,
		nodeName,
		owningTopologyName,
		owningTopology,
	)

	return deployment
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the final deployments for the given nodes.
func (r *DeploymentReconciler) RenderAll(
	owningTopology *clabernetesapisv1alpha1.Topology,
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
func (r *DeploymentReconciler) Conforms( //nolint: gocyclo
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

	if !clabernetesutilkubernetes.ExistingMapStringStringContainsAllExpectedKeyValues(
		existingDeployment.Spec.Template.Spec.NodeSelector,
		renderedDeployment.Spec.Template.Spec.NodeSelector,
	) {
		return false
	}

	if !reflect.DeepEqual(
		existingDeployment.Spec.Template.Spec.Tolerations,
		renderedDeployment.Spec.Template.Spec.Tolerations,
	) {
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

	if !clabernetesutilkubernetes.ExistingMapStringStringContainsAllExpectedKeyValues(
		existingDeployment.ObjectMeta.Annotations,
		renderedDeployment.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.ExistingMapStringStringContainsAllExpectedKeyValues(
		existingDeployment.ObjectMeta.Labels,
		renderedDeployment.ObjectMeta.Labels,
	) {
		return false
	}

	if !clabernetesutilkubernetes.ExistingMapStringStringContainsAllExpectedKeyValues(
		existingDeployment.Spec.Template.ObjectMeta.Annotations,
		renderedDeployment.Spec.Template.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.ExistingMapStringStringContainsAllExpectedKeyValues(
		existingDeployment.Spec.Template.ObjectMeta.Labels,
		renderedDeployment.Spec.Template.ObjectMeta.Labels,
	) {
		return false
	}

	if len(existingDeployment.ObjectMeta.OwnerReferences) != 1 {
		// we should have only one owner reference, the owning topology
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
	owningTopology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) {
	for nodeName, nodeConfig := range reconcileData.ResolvedConfigs {
		_, nodeExistedBefore := reconcileData.PreviousConfigs[nodeName]
		if !nodeExistedBefore {
			continue
		}

		if owningTopology.Spec.Connectivity == clabernetesconstants.ConnectivitySlurpeeth {
			determineNodeNeedsRestartSlurpeeth(reconcileData, nodeName)
		} else if !reflect.DeepEqual(nodeConfig, reconcileData.PreviousConfigs[nodeName]) {
			reconcileData.NodesNeedingReboot.Add(nodeName)
		}
	}
}

func determineNodeNeedsRestartSlurpeeth(
	reconcileData *ReconcileData,
	nodeName string,
) {
	previousConfig := reconcileData.PreviousConfigs[nodeName]
	currentConfig := reconcileData.ResolvedConfigs[nodeName]

	if previousConfig.Debug != currentConfig.Debug {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if previousConfig.Name != currentConfig.Name {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if !reflect.DeepEqual(previousConfig.Mgmt, currentConfig.Mgmt) {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if !reflect.DeepEqual(previousConfig.Prefix, currentConfig.Prefix) {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if !reflect.DeepEqual(previousConfig.Topology.Nodes, currentConfig.Topology.Nodes) {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if !reflect.DeepEqual(previousConfig.Topology.Kinds, currentConfig.Topology.Kinds) {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	if !reflect.DeepEqual(previousConfig.Topology.Defaults, currentConfig.Topology.Defaults) {
		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}

	// we know (because we set this) that topology will never be nil and links will always be slices
	// that are len 2... so we are a little risky here but its probably ok :)
	for idx := range previousConfig.Topology.Links {
		previousASide := previousConfig.Topology.Links[idx].Endpoints[0]
		currentASide := currentConfig.Topology.Links[idx].Endpoints[0]

		if previousASide == currentASide {
			// as long as "a" side is the same, slurpeeth will auto update itself since launcher is
			// watching the connectivity cr
			continue
		}

		reconcileData.NodesNeedingReboot.Add(nodeName)

		return
	}
}

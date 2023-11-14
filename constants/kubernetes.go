package constants

const (
	// KubernetesConfigMap is a const to use for "configmap".
	KubernetesConfigMap = "configmap"

	// KubernetesService is a const to use for "service".
	KubernetesService = "service"

	// KubernetesDeployment is a const to use for "deployment".
	KubernetesDeployment = "deployment"

	// KubernetesServiceClusterIPType is a const to use for "ClusterIP".
	KubernetesServiceClusterIPType = "ClusterIP"

	// KubernetesServiceLoadBalancerType is a const to use for "LoadBalancer".
	KubernetesServiceLoadBalancerType = "LoadBalancer"
)

const (
	// KubernetesCRIUnknown is a const for when we dont know what the CRI type is in a cluster.
	KubernetesCRIUnknown = "unknown"
	// KubernetesCRIContainerd is a const for the "containerd" type of CRI in a cluster.
	KubernetesCRIContainerd = "containerd"
	// KubernetesCRICrio is a const for the "cri-o" type of CRI in a cluster.
	KubernetesCRICrio = "crio"
)

const (
	// KubernetesCRISockContainerdPath is the path where the containerd sock lives.
	KubernetesCRISockContainerdPath = "/run/containerd"
	// KubernetesCRISockContainerd is the containerd sock filename.
	KubernetesCRISockContainerd = "containerd.sock"
)

const (
	// LauncherCRISockPath is the path where, if configured, the CRI sock is mounted in launcher
	// pods.
	LauncherCRISockPath = "/clabernetes/.node"
)

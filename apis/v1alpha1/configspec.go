package v1alpha1

import k8scorev1 "k8s.io/api/core/v1"

// ConfigMetadata holds "global" configuration data that will be applied to all objects created by
// the clabernetes controller.
type ConfigMetadata struct {
	// Annotations holds key/value pairs that should be set as annotations on clabernetes created
	// resources. Note that (currently?) there is no input validation here, but this data must be
	// valid kubernetes annotation data.
	// +optional
	Annotations map[string]string `json:"annotations"`
	// Labels holds key/value pairs that should be set as labels on clabernetes created resources.
	// Note that (currently?) there is no input validation here, but this data must be valid
	// kubernetes label data.
	// +optional
	Labels map[string]string `json:"labels"`
}

// ConfigDeployment holds "global" or "default" configurations related to clabernetes spawned
// deployments. In the future this will likely include more of the "normal" (topology-level)
// deployment configs (ex: persistence, or maybe files from url).
type ConfigDeployment struct {
	// Resources is a mapping of container lab kind -> type -> default resource settings. If a key
	// "default" is set at the map root *or* under the "kind" level, and there is no more explicit
	// match for kind/type, then that default value will be used (unless overridden at the topology
	// level which takes precedence).
	// +optional
	Resources map[string]map[string]*k8scorev1.ResourceRequirements `json:"resources"`
	// PrivilegedLauncher, when true, sets the launcher containers to privileged. By default, we do
	// our best to *not* need this/set this, and instead set only the capabilities we need, however
	// its possible that some containers launched by the launcher may need/want more capabilities,
	// so this flag exists for users to bypass the default settings and enable fully privileged
	// launcher pods.
	// +optional
	PrivilegedLauncher bool `json:"privilegedLauncher"`
	// ContainerlabDebug sets the `--debug` flag when invoking containerlab in the launcher pods.
	// This is disabled by default.
	// +optional
	ContainerlabDebug bool `json:"containerlabDebug"`
	// LauncherLogLevel sets the launcher clabernetes worker log level -- this overrides whatever
	// is set on the controllers env vars for this topology. Note: omitempty because empty str does
	// not satisfy enum of course.
	// +kubebuilder:validation:Enum=disabled;critical;warn;info;debug
	// +optional
	LauncherLogLevel string `json:"launcherLogLevel,omitempty"`
}

// ConfigImagePull holds configurations relevant to how clabernetes launcher pods handle pulling
// images.
type ConfigImagePull struct {
	// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
	// pods.
	// +optional
	InsecureRegistries InsecureRegistries `json:"insecureRegistries"`
	// PullThroughOverride allows for overriding the image pull through mode for this
	// particular topology.
	// +kubebuilder:validation:Enum=auto;always;never
	// +optional
	PullThroughOverride string `json:"pullThroughOverride,omitempty"`
	// PullSecrets allows for providing secret(s) to use when pulling the image. This is only
	// applicable *if* ImagePullThrough mode is auto or always. The secret is used by the launcher
	// pod to pull the image via the cluster CRI. The secret is *not* mounted to the pod, but
	// instead is used in conjunction with a job that spawns a pod using the specified secret. The
	// job will kill the pod as soon as the image has been pulled -- we do this because we don't
	// care if the pod runs, we only care that the image gets pulled on a specific node. Note that
	// just like "normal" pull secrets, the secret needs to be in the namespace that the topology
	// is in.
	// +listType=set
	// +optional
	PullSecrets []string `json:"pullSecrets"`
}

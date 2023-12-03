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
	// ResourcesDefault is the default set of resources for clabernetes launcher pods. This is used
	// only as a last option if a Topology does not have resources, and there are no resources for
	// the given containerlab kind/type
	// +optional
	ResourcesDefault *k8scorev1.ResourceRequirements `json:"resourcesDefault"`
	// ResourcesByContainerlabKind is a mapping of container lab kind -> type -> default resource
	// settings. Note that a key value of "default" in the inner map will apply the given resources
	// for any pod of that containerlab *kind*. For example:
	// {
	//   "srl": {
	//     "default": DEFAULT RESOURCES FOR KIND "srl",
	//     "ixr10": RESOURCES FOR KIND "srl", TYPE "ixr10"
	// }
	// Given resources as above, a containerlab node of kind "srl" and "type" ixr10" would get the
	// specific resources as allocated in the ixr10 key, whereas a containerlab kind of "srl" and
	// "type" unset or "ixr6" would get the "default" resource settings. To apply global default
	// resources, regardless of containerlab kind/type, use the `resourcesDefault` field.
	// +optional
	ResourcesByContainerlabKind map[string]map[string]*k8scorev1.ResourceRequirements `json:"resourcesByContainerlabKind"` //nolint:lll
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
	// PullThroughOverride allows for overriding the image pull through mode for this
	// particular topology.
	// +kubebuilder:validation:Enum=auto;always;never
	// +optional
	PullThroughOverride string `json:"pullThroughOverride,omitempty"`
}

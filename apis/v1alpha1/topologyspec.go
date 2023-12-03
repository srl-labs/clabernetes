package v1alpha1

import k8scorev1 "k8s.io/api/core/v1"

// FileFromConfigMap represents a file that you would like to mount (from a configmap) in the
// launcher pod for a given node.
type FileFromConfigMap struct {
	// FilePath is the path to mount the file.
	FilePath string `json:"filePath"`
	// ConfigMapName is the name of the configmap to mount.
	ConfigMapName string `json:"configMapName"`
	// ConfigMapPath is the path/key in the configmap to mount, if not specified the configmap will
	// be mounted without a sub-path.
	// +optional
	ConfigMapPath string `json:"configMapPath"`
}

// FileFromURL represents a file that you would like to mount from a URL in the launcher pod for
// a given node.
type FileFromURL struct {
	// FilePath is the path to mount the file.
	FilePath string `json:"filePath"`
	// URL is the url to fetch and mount at the provided FilePath. This URL must be a url that can
	// be simply downloaded and dumped to disk -- meaning a normal file server type endpoint or if
	// using GitHub or similar a "raw" path.
	URL string `json:"url"`
}

// Persistence holds information about how to persist the containlerab lab directory for each node
// in a topology.
type Persistence struct {
	// Enabled indicates if persistence of hte containerlab lab/working directory will be placed in
	// a mounted PVC.
	Enabled bool `json:"enabled"`
	// ClaimSize is the size of the PVC for this topology -- if not provided this defaults to 5Gi.
	// If provided, the string value must be a valid kubernetes storage requests style string. Note
	// the claim size *cannot be made smaller* once created, but it *can* be expanded. If you need
	// to make the claim smaller you must delete the topology (or the node from the topology) and
	// re-add it.
	// +optional
	ClaimSize string `json:"claimSize,omitempty"`
	// StorageClassName is the storage class to set in the PVC -- if not provided this will be left
	// empty which will end up using your default storage class. Note that currently we assume you
	// have (as default) or provide a dynamically provisionable storage class, hence no selector.
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
// pods.
type InsecureRegistries []string

// Definition holds the underlying topology definition for the Topology CR. A Topology *must* have
// one -- and only one -- definition type defined.
type Definition struct {
	// Containerlab holds a valid containerlab topology.
	// +optional
	Containerlab string `json:"containerlab,omitempty"`
	// Kne holds a valid kne topology.
	// +optional
	Kne string `json:"kne,omitempty"`
}

// Expose holds configurations relevant to how clabernetes exposes a topology.
type Expose struct {
	// DisableNodeAliasService indicates if headless services for each node in a containerlab
	// topology should *not* be created. By default, clabernetes creates these headless services for
	// each node so that "normal" docker and containerlab service discovery works -- this means you
	// can simply resolve "my-neat-node" from within the namespace of a topology like you would in
	// docker locally. You may wish to disable this feature though if you have no need of it and
	// just don't want the extra services around. Additionally, you may want to disable this feature
	// if you are running multiple labs in the same namespace (which is not generally recommended by
	// the way!) as you may end up in a situation where a name (i.e. "leaf1") is duplicated in more
	// than one topology -- this will cause some problems for clabernetes!
	// +optional
	DisableNodeAliasService bool `json:"disableNodeAliasService"`
	// DisableExpose indicates if exposing nodes via LoadBalancer service should be disabled, by
	// default any mapped ports in a containerlab topology will be exposed.
	// +optional
	DisableExpose bool `json:"disableExpose"`
	// DisableAutoExpose disables the automagic exposing of ports for a given topology. When this
	// setting is disabled clabernetes will not auto add ports so if you want to expose (via a
	// load balancer service) you will need to have ports outlined in your containerlab config
	// (or equivalent for kne). When this is `false` (default), clabernetes will add and expose the
	// following list of ports to whatever ports you have already defined:
	//
	// 21    - tcp - ftp
	// 22    - tcp - ssh
	// 23    - tcp - telnet
	// 80    - tcp - http
	// 161   - udp - snmp
	// 443   - tcp - https
	// 830   - tcp - netconf (over ssh)
	// 5000  - tcp - telnet for vrnetlab qemu host
	// 5900  - tcp - vnc
	// 6030  - tcp - gnmi (arista default)
	// 9339  - tcp - gnmi/gnoi
	// 9340  - tcp - gribi
	// 9559  - tcp - p4rt
	// 57400 - tcp - gnmi (nokia srl/sros default)
	//
	// This setting is *ignored completely* if `DisableExpose` is true!
	//
	// +optional
	DisableAutoExpose bool `json:"disableAutoExpose"`
}

// Deployment holds configurations relevant to how clabernetes configures deployments that make
// up a given topology.
type Deployment struct {
	// Resources is a mapping of nodeName (or "default") to kubernetes resource requirements -- any
	// value set here overrides the "global" config resource definitions. If a key "default" is set,
	// those resource values will be preferred over *all global settings* for this topology --
	// meaning, the "global" resource settings will never be looked up for this topology, and any
	// kind/type that is *not* in this resources map will have the "default" resources from this
	// mapping applied.
	// +optional
	Resources map[string]k8scorev1.ResourceRequirements `json:"resources"`
	// PrivilegedLauncher, when true, sets the launcher containers to privileged. By default, we do
	// our best to *not* need this/set this, and instead set only the capabilities we need, however
	// its possible that some containers launched by the launcher may need/want more capabilities,
	// so this flag exists for users to bypass the default settings and enable fully privileged
	// launcher pods. If this value is unset, the global config value (default of "false") will be
	// used.
	// +optional
	PrivilegedLauncher *bool `json:"privilegedLauncher"`
	// FilesFromConfigMap is a slice of FileFromConfigMap that define the configmap/path and node
	// and path on a launcher node that the file should be mounted to. If the path is not provided
	// the configmap is mounted in its entirety (like normal k8s things), so you *probably* want
	// to specify the sub path unless you are sure what you're doing!
	// +optional
	FilesFromConfigMap map[string][]FileFromConfigMap `json:"filesFromConfigMap"`
	// FilesFromURL is a mapping of FileFromURL that define a URL at which to fetch a file, and path
	// on a launcher node that the file should be downloaded to. This is useful for configs that are
	// larger than the ConfigMap (etcd) 1Mb size limit.
	// +optional
	FilesFromURL map[string][]FileFromURL `json:"filesFromURL"`
	// Persistence holds configurations relating to persisting each nodes working containerlab
	// directory.
	// +optional
	Persistence Persistence `json:"persistence"`
	// ContainerlabDebug sets the `--debug` flag when invoking containerlab in the launcher pods.
	// This is disabled by default. If this value is unset, the global config value (default of
	// "false") will be used.
	// +optional
	ContainerlabDebug *bool `json:"containerlabDebug"`
	// LauncherImage sets the default launcher image to use when spawning launcher deployments for
	// this Topology. This is optional, the launcher image will default to whatever is set in the
	// global config CR.
	// +optional
	LauncherImage string `json:"launcherImage,omitempty"`
	// LauncherImagePullPolicy sets the default launcher image pull policy to use when spawning
	// launcher deployments for this Topology. This is also optional and defaults to whatever is set
	// in the global config CR (typically "IfNotPresent"). Note: omitempty because empty str does
	// not satisfy enum of course.
	// +kubebuilder:validation:Enum=IfNotPresent;Always;Never
	// +optional
	LauncherImagePullPolicy string `json:"launcherImagePullPolicy,omitempty"`
	// LauncherLogLevel sets the launcher clabernetes worker log level -- this overrides whatever
	// is set on the controllers env vars for this topology. Note: omitempty because empty str does
	// not satisfy enum of course.
	// +kubebuilder:validation:Enum=disabled;critical;warn;info;debug
	// +optional
	LauncherLogLevel string `json:"launcherLogLevel,omitempty"`
}

// ImagePull holds configurations relevant to how clabernetes launcher pods handle pulling
// images.
type ImagePull struct {
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

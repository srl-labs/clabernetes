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
	// Mode sets the file permissions when mounting the configmap. Since the configmap will be read
	// only filesystem anyway, we basically just want to expose if the file should be mounted as
	// executable or not. So, default permissions would be 0o444 (read) and execute would be 0o555.
	// +kubebuilder:validation:Enum=read;execute
	// +kubebuilder:default=read
	Mode string `json:"mode"`
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
	// Scheduling holds information about how the launcher pod(s) should be configured with respect
	// to "scheduling" things (affinity/node selector/tolerations).
	// +optional
	Scheduling Scheduling `json:"scheduling"`
	// PrivilegedLauncher, when true, sets the launcher containers to privileged. Historically we
	// tried very hard to *not* need to set privileged mode on pods, however the reality is it is
	// much, much easier to get various network operating system images booting with this enabled,
	// so, the default mode is to set the privileged flag on pods. Disabling this option causes
	// clabernetes to try to run the pods for this topology in the "not so privileged" mode -- this
	// basically means we mount all capabilities we think should be available, set apparmor to
	// "unconfined", and mount paths like /dev/kvm and dev/net/tun. With this "not so privileged"
	// mode, Nokia SRL devices and Arista cEOS devices have been able to boot on some clusters, but
	// your mileage may vary. In short: if you don't care about having some privileged pods, just
	// leave this alone.
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
	// ContainerlabTimeout sets the `--timeout` flag when invoking containerlab in the launcher
	// pods.
	// +optional
	ContainerlabTimeout string `json:"containerlabTimeout"`
	// ContainerlabVersion sets a custom version to use for containerlab -- when set this will cause
	// the launcher pods to download and use this specific version of containerlab. Setting a bad
	// version (version that doesnt exist/typo/etc.) will cause pods to fail to launch, so be
	// careful! You never "need" to this as the publicly available launcher image will always be
	// built with a (reasonably) up to date containerlab version, this setting exists in case you
	// want to pin back to an older version for some reason or you want to be bleeding edge with
	// some new feature (but do note that just because it exists in containerlab doesnt
	// *necessarily* mean it will be auto-working in clabernetes!
	// +optional
	ContainerlabVersion string `json:"containerlabVersion,omitempty"`
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

// Scheduling holds information about how the launcher pod(s) should be configured with respect
// to "scheduling" things (affinity/node selector/tolerations).
type Scheduling struct {
	// NodeSelector sets the node selector that will be configured on all launcher pods for this
	// Topology.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Tolerations is a list of Tolerations that will be set on the launcher pod spec.
	// +listType=atomic
	// +optional
	Tolerations []k8scorev1.Toleration `json:"tolerations"`
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
	// DockerDaemonConfig allows for setting the docker daemon config for all launchers in this
	// topology. The secret *must be present in the namespace of this topology*. The secret *must*
	// contain a key "daemon.json" -- as this secret will be mounted to /etc/docker and docker will
	// be expecting the config at /etc/docker/daemon.json.
	// +optional
	DockerDaemonConfig string `json:"dockerDaemonConfig,omitempty"`
}

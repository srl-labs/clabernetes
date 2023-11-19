package v1alpha1

import (
	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TopologyCommonObject is an interface that clabernetes topology objects must implement -- it
// allows the CR object to be handled as a normal controller runtime object but with the addition
// of a method that allows for accessing the common embedded TopologyCommonSpec struct that is
// embedded in all clabernetes topology CR types.
type TopologyCommonObject interface {
	ctrlruntimeclient.Object
	GetTopologyCommonSpec() TopologyCommonSpec
	GetTopologyStatus() TopologyStatus
}

// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
// pods.
type InsecureRegistries []string

// LinkEndpointElementCount defines the expected element count for a link endpoint slice.
const LinkEndpointElementCount = 2

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
	// If provided, the string value must be a valid kubernetes storage requests style string.
	// +optional
	ClaimSize string `json:"claimSize"`
	// StorageClassName is the storage class to set in the PVC -- if not provided this will be left
	// empty which will end up using your default storage class.
	// +optional
	StorageClassName string `json:"storageClassName"`
}

// TopologyCommonSpec holds fields that are common across different CR types for their spec.
type TopologyCommonSpec struct {
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
	// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
	// pods.
	// +optional
	InsecureRegistries InsecureRegistries `json:"insecureRegistries"`
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
	// This is disabled by default.
	// +optional
	ContainerlabDebug bool `json:"containerlabDebug"`
	// LauncherLogLevel sets the launcher clabernetes worker log level -- this overrides whatever
	// is set on the controllers env vars for this topology. Note: omitempty because empty str does
	// not satisfy enum of course.
	// +kubebuilder:validation:Enum=disabled;critical;warn;info;debug
	// +optional
	LauncherLogLevel string `json:"launcherLogLevel,omitempty"`
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
	// launcher pods.
	// +optional
	PrivilegedLauncher bool `json:"privilegedLauncher"`
	// ImagePullThroughOverride allows for overriding the image pull through mode for this
	// particular topology.
	// +kubebuilder:validation:Enum=auto;always;never
	// +optional
	ImagePullThroughOverride string `json:"imagePullThroughOverride"`
}

// LinkEndpoint is a simple struct to hold node/interface name info for a given link.
type LinkEndpoint struct {
	// NodeName is the name of the node this link resides on.
	NodeName string `json:"nodeName"`
	// InterfaceName is the name of the interface on the node this link is on.
	InterfaceName string `json:"interfaceName"`
}

// Tunnel represents a VXLAN tunnel between clabernetes nodes (as configured by containerlab).
type Tunnel struct {
	// ID is the VXLAN ID (vnid) for the tunnel.
	ID int `json:"id"             yaml:"id"`
	// LocalNodeName is the name of the local node for this tunnel.
	LocalNodeName string `json:"localNodeName"  yaml:"localNodeName"`
	// RemoteName is the name of the service to contact the remote end of the tunnel.
	RemoteName string `json:"remoteName"     yaml:"remoteName"`
	// RemoteNodeName is the name of the remote node.
	RemoteNodeName string `json:"remoteNodeName" yaml:"remoteNodeName"`
	// LocalLinkName is the local link name for the local end of the tunnel.
	LocalLinkName string `json:"localLinkName"  yaml:"localLinkName"`
	// RemoteLinkName is the remote link name for the remote end of the tunnel.
	RemoteLinkName string `json:"remoteLinkName" yaml:"remoteLinkName"`
}

// ExposedPorts holds information about exposed ports.
type ExposedPorts struct {
	// LoadBalancerAddress holds the address assigned to the load balancer exposing ports for a
	// given node.
	LoadBalancerAddress string `json:"loadBalancerAddress"`
	// TCPPorts is a list of TCP ports exposed on the LoadBalancer service.
	// +listType=set
	TCPPorts []int `json:"tcpPorts"`
	// UDPPorts is a list of UDP ports exposed on the LoadBalancer service.
	// +listType=set
	UDPPorts []int `json:"udpPorts"`
}

// TopologyStatus is a common struct used inline as CR status for topology resources.
type TopologyStatus struct {
	// Configs is a map of node name -> clab config -- in other words, this is the original
	// containerlab configuration broken up and modified to use multi-node topology setup (via host
	// links+vxlan). This is stored as a raw message so we don't have any weirdness w/ yaml tags
	// instead of json tags in clab things, and so we kube builder doesnt poop itself.
	Configs string `json:"configs"`
	// ConfigsHash is a hash of the last stored Configs data.
	ConfigsHash string `json:"configsHash"`
	// Tunnels is a mapping of tunnels that need to be configured between nodes (nodes:[]tunnels).
	Tunnels map[string][]*Tunnel `json:"tunnels"`
	// TunnelsHash is a hash of the last stored Tunnels data. As this can change due to the dns
	// suffix changing and not just the config changing we need to independently track this state.
	TunnelsHash string `json:"tunnelsHash"`
	// FilesFromURLHashes is a mapping of node FilesFromURL hashes stored so we can easily identify
	// which nodes had changes in their FilesFromURL data so we can know to restart them.
	FilesFromURLHashes map[string]string `json:"filesFromURLHashes"`
	// NodeExposedPorts holds a map of (containerlab) nodes and their exposed ports
	// (via load balancer).
	NodeExposedPorts map[string]*ExposedPorts `json:"nodeExposedPorts"`
	// NodeExposedPortsHash is a hash of the last stored NodeExposedPorts data.
	NodeExposedPortsHash string `json:"nodeExposedPortsHash"`
}

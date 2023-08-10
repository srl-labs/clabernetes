package v1alpha1

// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
// pods.
type InsecureRegistries []string

// LinkEndpointElementCount defines the expected element count for a link endpoint slice.
const LinkEndpointElementCount = 2

// FileFromConfigMap represents a file that you would like to mount (from a configmap) in the
// launcher pod for a given node.
type FileFromConfigMap struct {
	// NodeName is the name of the node (as in node from the clab topology) that the file should
	// be mounted for.
	NodeName string `json:"nodeName"`
	// FilePath is the path to mount the file.
	FilePath string `json:"filePath"`
	// ConfigMapName is the name of the configmap to mount.
	ConfigMapName string `json:"configMapName"`
	// ConfigMapPath is the path/key in the configmap to mount, if not specified the configmap will
	// be mounted without a sub-path.
	// +optional
	ConfigMapPath string `json:"configMapPath"`
}

// TopologyCommonSpec holds fields that are common across different CR types for their spec.
type TopologyCommonSpec struct {
	// DisableExpose indicates if exposing nodes via LoadBalancer service should be disabled, by
	// default any mapped ports in a containerlab topology will be exposed.
	// +optional
	DisableExpose bool `json:"disableExpose"`
	// InsecureRegistries is a slice of strings of insecure registries to configure in the launcher
	// pods.
	// +optional
	InsecureRegistries InsecureRegistries `json:"insecureRegistries"`
	// FilesFromConfigMap is a slice of FileFromConfigMap that define the configmap/path and node
	// and path on a launcher node that the file should be mounted to. If the path is not provided
	// the configmap is mounted in its entirety (like normal k8s things), so you *probably* want
	// to specify the sub path unless you are sure what you're doing!
	// +listType=atomic
	// +optional
	FilesFromConfigMap []FileFromConfigMap `json:"filesFromConfigMap"`
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
	ID int `json:"id"`
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
	// NodeExposedPorts holds a map of (containerlab) nodes and their exposed ports
	// (via load balancer).
	NodeExposedPorts map[string]*ExposedPorts `json:"nodeExposedPorts"`
	// NodeExposedPortsHash is a hash of the last stored NodeExposedPorts data.
	NodeExposedPortsHash string `json:"nodeExposedPortsHash"`
}

package v1alpha1

// LinkEndpointElementCount defines the expected element count for a link endpoint slice.
const LinkEndpointElementCount = 2

// ReconcileHashes holds hashes of the last recorded reconciliation -- these are used to know if
// things have changed between the last and current reconciliation.
type ReconcileHashes struct {
	// Config is the last stored hash of the rendered config(s) -- that is, the map of "sub
	// topologies" representing the overall Topology.Spec.Definition.
	Config string `json:"config"`
	// Tunnels is the last stored hash of the tunnels that provided connectivity between the
	// launcher nodes. As this can change due to the dns suffix changing and not just the config
	// changing we need to independently track this state.
	Tunnels string `json:"tunnels"`
	// ExposedPorts is the last stored hash of the exposed ports mapping for this Topology. Note
	// that while we obviously care about the exposed ports on a *per node basis*, we don't need to
	// track that here -- this is here strictly to track differences in the load balancer service --
	// the actual sub-topologies (or sub-configs) effectively track the expose port status per node.
	ExposedPorts string `json:"exposedPorts"`
	// FilesFromURL is the hash of the last stored mapping of files from URL (to node mapping). Note
	// that this is tracked on a *per node basis* because the URL of a file could be updated without
	// any change to the actual config/topology (or sub-config/sub-topology); as such we need to
	// explicitly track this per node to know when a node needs to be restarted such that the new
	// URL is "picked up" by the node/launcher.
	FilesFromURL map[string]string `json:"filesFromURL"`
	// ImagePullSecrets is the hash of hte last stored image pull secrets for this Topology.
	ImagePullSecrets string `json:"imagePullSecrets"`
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

package v1alpha1

// PointToPointTunnel holds information necessary for creating a tunnel between two interfaces on
// different nodes of a clabernetes Topology. This connection can be established by using clab tools
// (vxlan) or the experimental slurpeeth (tcp tunnel magic).
type PointToPointTunnel struct {
	// TunnelID is the id number of the tunnel (vnid or segment id).
	TunnelID int `json:"tunnelID"`
	// Destination is the destination service to connect to (qualified k8s service name).
	Destination string `json:"destination"`
	// LocalNodeName is the name (in the clabernetes topology) of the local node for this side of
	// the tunnel.
	LocalNode string `json:"localNode"`
	// LocalInterface is the local termination of this tunnel.
	LocalInterface string `json:"localInterface"`
	// RemoteNode is the name (in the clabernetes topology) of the remote node for this side of the
	// tunnel.
	RemoteNode string `json:"remoteNode"`
	// RemoteInterface is the remote termination interface of this tunnel -- necessary to store so
	// can properly align tunnels (and ids!) between nodes; basically to know which tunnels are
	// "paired up".
	RemoteInterface string `json:"remoteInterface"`
}

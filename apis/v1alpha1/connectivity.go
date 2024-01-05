package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Connectivity is an object that holds information about a connectivity between launcher pods in
// a clabernetes Topology.
// +k8s:openapi-gen=true
type Connectivity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectivitySpec   `json:"spec,omitempty"`
	Status ConnectivityStatus `json:"status,omitempty"`
}

// ConnectivitySpec is the spec for a Connectivity resource.
type ConnectivitySpec struct {
	// PointToPointTunnels holds point-to-point connectivity information for a given topology. The
	// mapping is nodeName (i.e. srl1) -> p2p tunnel data. Both sides of the tunnel should be able
	// to use this information to establish connectivity between Topology nodes.
	PointToPointTunnels map[string][]*PointToPointTunnel `json:"pointToPointTunnels"`
}

// ConnectivityStatus is the status for a Connectivity resource.
type ConnectivityStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConnectivityList is a list of Connectivity objects.
type ConnectivityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Connectivity `json:"items"`
}

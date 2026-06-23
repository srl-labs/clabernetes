package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Link represents a single point-to-point link between two Topology nodes that live in different
// launcher pods -- i.e. a cross-pod link realized as a vxlan or slurpeeth tunnel. Links that are
// internal to a node group (nodes sharing one network namespace) are NOT represented as Link
// objects; they remain inside the owning Node's sub-topology.
//
// A Link is created and owned by the Topology controller as part of the decomposed (scalable)
// reconcile path; see docs/design/0001-scale-node-link-crds.md. Representing each link as its own
// object is what removes the per-topology Connectivity object as a scaling bottleneck.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=links,shortName=clablink
// +kubebuilder:printcolumn:JSONPath=".spec.topologyName",name=Topology,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.tunnelID",name=TunnelID,type=integer
// +kubebuilder:printcolumn:JSONPath=".status.ready",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
type Link struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinkSpec   `json:"spec,omitempty"`
	Status LinkStatus `json:"status,omitempty"`
}

// LinkSpec is the spec for a Link resource.
type LinkSpec struct {
	// TopologyName is the name of the Topology that owns this Link.
	TopologyName string `json:"topologyName"`
	// EndpointA is one end of the link.
	EndpointA LinkEndpoint `json:"endpointA"`
	// EndpointB is the other end of the link. Its NodeName may be the containerlab "host" keyword
	// for host links.
	EndpointB LinkEndpoint `json:"endpointB"`
	// Connectivity is the connectivity flavor (vxlan or slurpeeth) used to realize this link.
	// +kubebuilder:validation:Enum=vxlan;slurpeeth
	// +optional
	Connectivity string `json:"connectivity,omitempty"`
	// TunnelID is the vxlan VNI / slurpeeth segment id allocated to this link. It is assigned once
	// by the owning Topology and is stable for the life of the Link -- renumbering a live link would
	// tear down a working tunnel. A value of 0 means an id has not yet been allocated.
	// +optional
	TunnelID int `json:"tunnelID"`
}

// LinkStatus is the status for a Link resource.
type LinkStatus struct {
	// Ready indicates whether both endpoints of the link have established connectivity.
	Ready bool `json:"ready"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LinkList is a list of Link objects.
type LinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Link `json:"items"`
}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Topology is an object that holds information about a clabernetes Topology -- that is, a valid
// topology file (ex: containerlab topology), and any associated configurations.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="topologies"
// +kubebuilder:printcolumn:JSONPath=".status.kind",name=Kind,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
// +kubebuilder:printcolumn:JSONPath=".status.topologyReady",name=Ready,type=boolean
type Topology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TopologySpec   `json:"spec,omitempty"`
	Status TopologyStatus `json:"status,omitempty"`
}

// TopologySpec is the spec for a Topology resource.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.naming) || has(self.naming)", message="naming is required once set"
type TopologySpec struct {
	// Definition defines the actual set of nodes (network ones, not k8s ones!) that this Topology
	// CR represents. Historically, and probably most often, this means Topology holds a "normal"
	// containerlab topology file that will be "clabernetsified", however this could also be a "kne"
	// config, or perhaps others in the future.
	Definition Definition `json:"definition"`
	// Expose holds configurations relevant to how clabernetes exposes a topology.
	// +optional
	Expose Expose `json:"expose"`
	// Deployment holds configurations relevant to how clabernetes configures deployments that make
	// up a given topology.
	// +optional
	Deployment Deployment `json:"deployment"`
	// StatusProbes holds the configurations relevant to how clabernetes and the launcher handle
	// checking and reporting the containerlab node status
	// +optional
	StatusProbes StatusProbes `json:"statusProbes"`
	// ImagePull holds configurations relevant to how clabernetes launcher pods handle pulling
	// images.
	// +optional
	ImagePull ImagePull `json:"imagePull"`
	// Naming tells the clabernetes controller how it should name resources it creates -- that is
	// whether it should include the containerlab topology name as a prefix on resources spawned
	// from this Topology or not; this includes the actual (containerlab) node Deployment(s), as
	// well as the Service(s) for the Topology. This setting has three modes; "prefixed" -- which of
	// course includes the containerlab topology name as a prefix, "non-prefixed" which does *not*
	// include the containerlab topology name as a prefix, and "global" which defers to the global
	// config setting for this (which defaults to "prefixed").
	// "non-prefixed" mode should only be enabled when/if Topologies are deployed in their own
	// namespace -- the reason for this is simple: if two Topologies exist in the same namespace
	// with a (containerlab) node named "my-router" there will be a conflicting Deployment and
	// Services for the "my-router" (containerlab) node. Note that this field is immutable! If you
	// want to change its value you need to delete the Topology and re-create it.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="naming field is immutable, to change this value delete and re-create the Topology"
	// +kubebuilder:validation:Enum=prefixed;non-prefixed;global
	// +kubebuilder:default=global
	Naming string `json:"naming"`
	// Connectivity defines the type of connectivity to use between nodes in the topology. The
	// default behavior is to use vxlan tunnels, alternatively you can enable a more experimental
	// "slurpeeth" connectivity flavor that stuffs traffic into tcp tunnels to avoid any vxlan mtu
	// and/or fragmentation challenges.
	// +kubebuilder:validation:Enum=vxlan;slurpeeth
	// +kubebuilder:default=vxlan
	Connectivity string `json:"connectivity"`
}

// TopologyStatus is the status for a Topology resource.
type TopologyStatus struct {
	// Kind is the topology kind this CR represents -- for example "containerlab".
	// +kubebuilder:validation:Enum=containerlab;kne
	Kind string `json:"kind"`
	// RemoveTopologyPrefix holds the "resolved" value of the RemoveTopologyPrefix field -- that is
	// if it is unset (nil) when a Topology is created, the controller will use the default global
	// config value (false); if the field is non-nil, this status field will hold the non-nil value.
	RemoveTopologyPrefix *bool `json:"removeTopologyPrefix"`
	// ReconcileHashes holds the hashes form the last reconciliation run.
	ReconcileHashes ReconcileHashes `json:"reconcileHashes"`
	// Configs is a map of node name -> containerlab config -- in other words, this is the original
	// Topology.Spec.Definition converted to containerlab "sub-topologies" The actual
	// "sub-topologies"/"sub-configs" are stored as a string -- this is the actual containerlab
	// topology that gets mounted in the launcher pod.
	Configs map[string]string `json:"configs"`
	// ExposedPorts holds a map of (containerlab not k8s!) nodes and their exposed ports
	// (via load balancer).
	ExposedPorts map[string]*ExposedPorts `json:"exposedPorts"`
	// NodeReadiness is a map of nodename to readiness status. The readiness status is as reported
	// by the k8s startup/readiness probe (which is in turn managed by the status probe
	// configuration of the topology). The possible values are "notready" and "ready", "unknown".
	NodeReadiness map[string]string `json:"nodeReadiness"`
	// Conditions is a list of conditions for the topology custom resource.
	Conditions []metav1.Condition `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TopologyList is a list of Topology objects.
type TopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Topology `json:"items"`
}

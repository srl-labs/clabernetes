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
type Topology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TopologySpec   `json:"spec,omitempty"`
	Status TopologyStatus `json:"status,omitempty"`
}

// TopologySpec is the spec for a Topology resource.
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
	// ImagePull holds configurations relevant to how clabernetes launcher pods handle pulling
	// images.
	// +optional
	ImagePull ImagePull `json:"imagePull"`
}

// TopologyStatus is the status for a Containerlab topology resource.
type TopologyStatus struct {
	// Kind is the topology kind this CR represents -- for example "containerlab".
	// +kubebuilder:validation:Enum=containerlab;kne
	Kind string `json:"kind"`
	// ReconcileHashes holds the hashes form the last reconciliation run.
	ReconcileHashes ReconcileHashes `json:"reconcileHashes"`
	// Configs is a map of node name -> clab config -- in other words, this is the original
	// Topology.Spec.Definition broken up and modified to use multi-node topology setup (via host
	// links+vxlan). This is stored as a raw message so we don't have any weirdness w/ yaml tags
	// instead of json tags in clab things, and so we kube builder doesnt poop itself.
	Configs string `json:"configs"`
	// NodeExposedPorts holds a map of (containerlab) nodes and their exposed ports
	// (via load balancer).
	NodeExposedPorts map[string]*ExposedPorts `json:"nodeExposedPorts"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TopologyList is a list of Topology objects.
type TopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Topology `json:"items"`
}

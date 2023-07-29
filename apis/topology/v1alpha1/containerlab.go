package v1alpha1

import (
	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Containerlab represents a "normal" containerlab topology file.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="containerlabs"
type Containerlab struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerlabSpec   `json:"spec,omitempty"`
	Status ContainerlabStatus `json:"status,omitempty"`
}

// ContainerlabSpec is the spec for a Containerlab topology resource.
type ContainerlabSpec struct {
	// Config is a "normal" containerlab configuration file.
	Config string `json:"config"`
}

// ContainerlabStatus is the status for a Containerlab topology resource.
type ContainerlabStatus struct {
	// Configs is a map of node name -> clab config -- in other words, this is the original
	// containerlab configuration broken up and modified to use multi-node topology setup (via host
	// links+vxlan). This is stored as a raw message so we don't have any weirdness w/ yaml tags
	// instead of json tags in clab things, and so we kube builder doesnt poop itself on it.
	Configs string `json:"configs"`
	// ConfigsHash is a hash of the last storedConfgs data.
	ConfigsHash string `json:"configsHash"`
	// Tunnels is a mapping of tunnels that need to be configured between nodes (nodes:[]tunnels).
	Tunnels map[string][]*clabernetesapistopology.Tunnel `json:"tunnels"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerlabList is a list of Containerlab topology objects.
type ContainerlabList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Containerlab `json:"items"`
}

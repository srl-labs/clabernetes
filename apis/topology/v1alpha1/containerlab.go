package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Containerlab is an object that holds a "normal" containerlab topology file and any additional
// data necessary to "clabernetes-ify" it.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="containerlabs"
type Containerlab struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerlabSpec   `json:"spec,omitempty"`
	Status ContainerlabStatus `json:"status,omitempty"`
}

// GetTopologyCommonSpec returns the Containerlab resource's TopologyCommonSpec embedded in the
// object's spec.
func (c *Containerlab) GetTopologyCommonSpec() TopologyCommonSpec {
	return c.Spec.TopologyCommonSpec
}

// GetTopologyStatus returns the Containerlab resource's TopologyStatus embedded in the object's
// status.
func (c *Containerlab) GetTopologyStatus() TopologyStatus {
	return c.Status.TopologyStatus
}

// SetTopologyStatus sets Kne resource's TopologyStatus embedded in the object's status.
func (c *Containerlab) SetTopologyStatus(s TopologyStatus) { //nolint:gocritic
	c.Status.TopologyStatus = s
}

// ContainerlabSpec is the spec for a Containerlab topology resource.
type ContainerlabSpec struct {
	TopologyCommonSpec `json:",inline"`
	// Config is a "normal" containerlab configuration file.
	Config string `json:"config"`
}

// ContainerlabStatus is the status for a Containerlab topology resource.
type ContainerlabStatus struct {
	TopologyStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerlabList is a list of Containerlab topology objects.
type ContainerlabList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Containerlab `json:"items"`
}

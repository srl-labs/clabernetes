package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kne is an object that holds a "normal" kne topology file and any additional data necessary to
// "clabernetes-ify" it.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="knes"
type Kne struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KneSpec   `json:"spec,omitempty"`
	Status KneStatus `json:"status,omitempty"`
}

// GetTopologyCommonSpec returns the Kne resource's TopologyCommonSpec embedded in the object's
// spec.
func (c *Kne) GetTopologyCommonSpec() TopologyCommonSpec {
	return c.Spec.TopologyCommonSpec
}

// GetTopologyStatus returns the Kne resource's TopologyStatus embedded in the object's status.
func (c *Kne) GetTopologyStatus() TopologyStatus {
	return c.Status.TopologyStatus
}

// SetTopologyStatus sets Kne resource's TopologyStatus embedded in the object's status.
func (c *Kne) SetTopologyStatus(s TopologyStatus) {
	c.Status.TopologyStatus = s
}

// KneSpec is the spec for a Kne topology resource.
type KneSpec struct {
	TopologyCommonSpec `json:",inline"`
	// Topology is a "normal" kne topology proto file.
	Topology string `json:"topology"`
}

// KneStatus is the status for a Kne topology resource.
type KneStatus struct {
	TopologyStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KneList is a list of Kne topology objects.
type KneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Kne `json:"items"`
}

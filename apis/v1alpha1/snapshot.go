package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Snapshot is an object that represents a saved configuration snapshot for a Topology.
// A Snapshot is created when the clabernetes/snapshotRequested annotation is set on a Topology,
// and it stores the running configurations of all nodes in a backing ConfigMap.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path="snapshots"
// +kubebuilder:printcolumn:JSONPath=".spec.topologyRef",name=Topology,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
// +kubebuilder:printcolumn:JSONPath=".status.configMapRef",name=ConfigMap,type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",name=Phase,type=string
type Snapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotSpec   `json:"spec,omitempty"`
	Status SnapshotStatus `json:"status,omitempty"`
}

// SnapshotSpec is the spec for a Snapshot resource.
type SnapshotSpec struct {
	// TopologyRef is the name of the Topology to snapshot.
	TopologyRef string `json:"topologyRef"`
	// TopologyNamespace is the namespace of the referenced Topology.
	// +optional
	TopologyNamespace string `json:"topologyNamespace,omitempty"`
}

// SnapshotStatus is the status for a Snapshot resource.
type SnapshotStatus struct {
	// ConfigMapRef is the name of the ConfigMap holding saved node configs.
	// +optional
	ConfigMapRef string `json:"configMapRef,omitempty"`
	// Timestamp is when the snapshot was completed (RFC3339 format).
	// +optional
	Timestamp string `json:"timestamp,omitempty"`
	// NodeConfigs maps node name to list of saved file paths stored in the ConfigMap.
	// +optional
	NodeConfigs map[string][]string `json:"nodeConfigs,omitempty"`
	// Phase is the current lifecycle phase: Pending, Running, Completed, Failed.
	// +optional
	Phase string `json:"phase,omitempty"`
	// Message holds any error or informational message.
	// +optional
	Message string `json:"message,omitempty"`
}

const (
	// SnapshotPhasePending is the initial phase of a Snapshot.
	SnapshotPhasePending = "Pending"
	// SnapshotPhaseRunning indicates the snapshot is in progress.
	SnapshotPhaseRunning = "Running"
	// SnapshotPhaseCompleted indicates the snapshot completed successfully.
	SnapshotPhaseCompleted = "Completed"
	// SnapshotPhaseFailed indicates the snapshot failed.
	SnapshotPhaseFailed = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnapshotList is a list of Snapshot objects.
type SnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Snapshot `json:"items"`
}

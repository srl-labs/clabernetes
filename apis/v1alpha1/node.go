package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Node represents a single containerlab node -- or a group of nodes that share a network namespace
// (for example a Nokia SR-SIM "distributed" node declared with network-mode: container:<name>) --
// that belongs to a Topology. It carries that node's single-node containerlab "sub-topology" plus
// the per-node configuration needed to launch it.
//
// A Node is created and owned by the Topology controller as part of the decomposed (scalable)
// reconcile path; see docs/design/0001-scale-node-link-crds.md. Because every per-node piece of
// state lives in its own Node object (rather than in one ever-growing field on the Topology), a
// Topology can describe thousands of nodes without hitting the etcd object size limit.
//
// NOTE: the plural/CLI name for this resource ("nodes") overlaps with the core Kubernetes Node at
// the kubectl level -- the fully qualified group (clabernetes.containerlab.dev) keeps it
// unambiguous to the API server, and the "clabnode" short name is provided so operators can select
// it without ambiguity. The final plural/short name is pending maintainer ratification.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=nodes,shortName=clabnode
// +kubebuilder:printcolumn:JSONPath=".spec.topologyName",name=Topology,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.kind",name=Kind,type=string
// +kubebuilder:printcolumn:JSONPath=".status.ready",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
}

// NodeSpec is the spec for a Node resource.
type NodeSpec struct {
	// TopologyName is the name of the Topology that owns this Node. It matches the owning Topology's
	// name and is also recorded as a label for selection.
	TopologyName string `json:"topologyName"`
	// NodeName is the name of the (containerlab) node this resource represents. For a node group
	// that shares a network namespace this is the name of the group's primary node.
	NodeName string `json:"nodeName"`
	// Kind is the topology kind this node was derived from -- "containerlab" or "kne".
	// +kubebuilder:validation:Enum=containerlab;kne
	Kind string `json:"kind"`
	// Definition is the single-node containerlab sub-topology (YAML) for this node. This is exactly
	// the content mounted into the launcher pod at /clabernetes/topo.clab.yaml today. Its size is
	// bounded by the node's own configuration and link degree -- never by the size of the overall
	// Topology -- which is what allows a Topology to scale to very large node counts.
	Definition string `json:"definition"`
	// Connectivity is the connectivity flavor (vxlan or slurpeeth) inherited from the owning
	// Topology, recorded here so the Node can be reconciled independently.
	// +kubebuilder:validation:Enum=vxlan;slurpeeth
	// +optional
	Connectivity string `json:"connectivity,omitempty"`
	// FilesFromConfigMap is the set of files mounted from ConfigMaps for this node -- this is how
	// bind-mounted config files (for example an frr.conf, a daemons file, or a startup-config) reach
	// the launcher. It is the per-node slice of the owning Topology's deployment.filesFromConfigMap.
	// The file *contents* live in their own ConfigMap object(s); this field only references them, so
	// even large per-node config sets never bloat the Node object.
	// +optional
	// +listType=atomic
	FilesFromConfigMap []FileFromConfigMap `json:"filesFromConfigMap,omitempty"`
	// FilesFromURL is the set of files (fetched from a URL) to mount for this node. This is the
	// per-node slice of the owning Topology's deployment.filesFromURL. URLs are used for files that
	// exceed the ConfigMap (~1MB) size limit.
	// +optional
	// +listType=atomic
	FilesFromURL []FileFromURL `json:"filesFromURL,omitempty"`
}

// NodeStatus is the status for a Node resource.
type NodeStatus struct {
	// Ready indicates whether this node has reported ready (via the launcher status probes).
	Ready bool `json:"ready"`
	// Readiness is the last reported readiness string for the node -- one of "ready", "notready",
	// or "unknown".
	// +optional
	Readiness string `json:"readiness,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeList is a list of Node objects.
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Node `json:"items"`
}

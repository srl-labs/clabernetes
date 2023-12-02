package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config is an object that holds global clabernetes config information. Note that this CR is
// expected to effectively be a global singleton -- that is, there should be only *one* of these,
// and it *must* be named `clabernetes` -- CRD metadata spec will enforce this (via x-validation
// rules).
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:validation:XValidation:rule=(self.metadata.name == 'clabernetes')
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// ConfigSpec is the spec for a Config resource.
type ConfigSpec struct {
	// InClusterDNSSuffix overrides the default in cluster dns suffix used when resolving services.
	// +optional
	InClusterDNSSuffix string `json:"inClusterDNSSuffix,omitempty"`
	// ConfigurationMergeMode defines how configmap configuration data is merged into this global
	// configuration object. This exists because when deploying clabernetes via helm, config data
	// is first deployed as a configmap, which is then loaded via the init container(s) and merged
	// back into this global singleton CR. This flag will be present in helm created configmap and
	// this CR -- if present in both locations this CR's value takes precedence. A value of "merge"
	// means that any value in the CR already will be preserved, while any value not in the CR will
	// be copied from the configmap and set here. A value of "replace" means that the values in the
	// configmap will replace any values in the CR.
	// +kubebuilder:validation:Enum=merge;replace
	// +kubebuilder:default=merge
	ConfigurationMergeMode string `json:"configurationMergeMode"`
	// Metadata holds "global" metadata -- that is, metadata that is applied to all objects created
	// by the clabernetes controller.
	// +optional
	Metadata ConfigMetadata `json:"metadata"`
	// Deployment holds clabernetes deployment related configuration settings.
	// +optional
	Deployment ConfigDeployment `json:"deployment"`
	// ImagePull holds configurations relevant to how clabernetes launcher pods handle pulling
	// images.
	// +optional
	ImagePull ConfigImagePull `json:"imagePull"`
}

// ConfigStatus is the status for a Config resource.
type ConfigStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigList is a list of Config objects.
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Config `json:"items"`
}

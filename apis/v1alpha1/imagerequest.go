package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageRequest is an object that represents a request (from a launcher pod) to pull an image on a
// given kubernetes node such that the image can be "pulled through" into the launcher docker
// daemon.
// +k8s:openapi-gen=true
type ImageRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageRequestSpec   `json:"spec,omitempty"`
	Status ImageRequestStatus `json:"status,omitempty"`
}

// ImageRequestSpec is the spec for a Config resource.
type ImageRequestSpec struct {
	// TopologyName is the name of the topology requesting the image.
	TopologyName string `json:"topologyName"`
	// TopologyNodeName is the name of the node in the topology (i.e. the router name in a
	// containerlab topology) that the image is being requested for.
	TopologyNodeName string `json:"topologyNodeName"`
	// KubernetesNode is the node where the launcher pod is running and where the image should be
	// pulled too.
	KubernetesNode string `json:"kubernetesNode"`
	// RequestedImage is the image that the launcher pod wants the controller to get pulled onto
	// the specified node.
	RequestedImage string `json:"requestedImage"`
	// RequestedImagePullSecrets is a list of configured pull secrets to set in the pull pod spec.
	// +listType=set
	// +optional
	RequestedImagePullSecrets []string `json:"requestedImagePullSecrets"`
}

// ImageRequestStatus is the status for a ImageRequest resource.
type ImageRequestStatus struct {
	// Accepted indicates that the ImageRequest controller has seen this image request and is going
	// to process it. This can be useful to let the requesting pod know that "yep, this is in the
	// works, and i can go watch the cri images on this node now".
	Accepted bool `json:"accepted"`
	// Complete indicates that the ImageRequest controller has seen that the puller pod has done its
	// job and that the image has been pulled onto the requested node.
	Complete bool `json:"complete"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageRequestList is a list of ImageRequest objects.
type ImageRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ImageRequest `json:"items"`
}

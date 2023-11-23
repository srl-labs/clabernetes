package types

// ImageRequest represents a request from a launcher pod to have an image pulled onto the node that
// the launcher is running on.
type ImageRequest struct {
	TopologyName          string   `json:"topologyName"`
	TopologyNamespace     string   `json:"topologyNamespace"`
	TopologyNodeName      string   `json:"topologyNodeName"`
	KubernetesNodeName    string   `json:"nodeName"`
	RequestingPodName     string   `json:"requestingPodName"`
	RequestedImageName    string   `json:"requestedImageName"`
	ConfiguredPullSecrets []string `json:"configuredPullSecrets"`
}

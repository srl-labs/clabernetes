package constants

const (
	// LabelApp is the label key for the simple app name.
	LabelApp = "clabernetes/app"

	// LabelName is the label key for the name of the project/application.
	LabelName = "clabernetes/name"

	// LabelComponent is the label key for the component label, it should define the component/tier
	// in the app, i.e. "manager".
	LabelComponent = "clabernetes/component"

	// LabelTopologyOwner is the label indicating the topology that owns the given resource.
	LabelTopologyOwner = "clabernetes/topologyOwner"

	// LabelTopologyNode is the label indicating the node the deployment represents in a topology.
	LabelTopologyNode = "clabernetes/topologyNode"
)

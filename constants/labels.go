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

	// LabelTopologyKind is the label indicating the resource *kind* the object is associated with.
	// For example, a "containerlab" kind, or a "kne" kind.
	LabelTopologyKind = "clabernetes/topologyKind"

	// LabelTopologyServiceType is a label that identifies what flavor of service a given service
	// is -- that is, it is either a "connectivity" service, or an "expose" service; note that
	// this is strictly a clabernetes concept, obviously not a kubernetes one!
	LabelTopologyServiceType = "clabernetes/topologyServiceType"

	LabelKRMKind = "clabernetes/labelkind"
)

const (
	// TopologyServiceTypeFabric is one of the allowed values for the LabelTopologyServiceType label
	// type -- this indicates that this service is of the type that facilitates the connectivity
	// between containerlab devices in the cluster.
	TopologyServiceTypeFabric = "fabric"
	// TopologyServiceTypeExpose is one of the allowed values for the LabelTopologyServiceType label
	// type -- this indicates that this service is of the type that is used for exposing ports on
	// a containerlab node via a LoadBalancer service.
	TopologyServiceTypeExpose = "expose"

	LabelKRMKindStartupConfig = "startup-config"
	LabelKRMKindExtraFiles    = "extra-files"
)

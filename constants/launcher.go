package constants

const (
	// ImagePullThroughModeAlways a constant representing the "always" image pull through mode for
	// the launcher pods.
	ImagePullThroughModeAlways = "always"

	// ImagePullThroughModeNever a constant representing the "never" image pull through mode for
	// the launcher pods.
	ImagePullThroughModeNever = "never"

	// ImagePullThroughModeAuto a constant representing the "auto" image pull through mode for
	// the launcher pods.
	ImagePullThroughModeAuto = "auto"

	// NamingModePrefixed is a constant representing the "prefixed" enum(ish) value for the naming
	// field of a Topology.
	NamingModePrefixed = "prefixed"

	// NamingModeNonPrefixed is a constant representing the "non-prefixed" enum(ish) value for the
	// naming field of a Topology.
	NamingModeNonPrefixed = "non-prefixed"

	// NamingModeGlobal is a constant representing the (default) "global" enum(ish) value for the
	// naming field of a Topology.
	NamingModeGlobal = "global"

	// ConnectivityVXLAN is a constant for the vxlan connectivity flavor.
	ConnectivityVXLAN = "vxlan"

	// ConnectivitySlurpeeth is a constant for the slurpeeth connectivity flavor.
	ConnectivitySlurpeeth = "slurpeeth"

	// NodeStatusFile is the file we write the node status to for launchers -- this is also used
	// by the deployment for startup/liveness probes.
	NodeStatusFile = "/clabernetes/.nodestatus"

	// NodeStatusHealthy is the content of hte NodeStatuesFile when/if the node in the launcher is
	// healthy.
	NodeStatusHealthy = "healthy"

	// NodeStatusReady is reported in the topology.status.nodereadiness map for nodes that have
	// their startup/readiness probes in a succeeding state.
	NodeStatusReady = "ready"

	// NodeStatusNotReady is reported in the topology.status.nodereadiness map for nodes that have
	// deployments running but do not report ready (via startup/readiness probes).
	NodeStatusNotReady = "notready"

	// NodeStatusUnknown is reported in the topology.status.nodereadiness map for nodes that have
	// no deployment available for whatever reason.
	NodeStatusUnknown = "unknown"

	// NodeStatusDeploymentDisabled is reported in the topology.status.nodereadiness map when the
	// parent topology has the "clabernetes/disableDeployments" label set.
	NodeStatusDeploymentDisabled = "deploymentDisabled"

	// TopologyStateDeploying is reported in the topology.status.topologyState field when the
	// topology is being deployed (not all nodes are ready yet).
	TopologyStateDeploying = "deploying"

	// TopologyStateRunning is reported in the topology.status.topologyState field when all nodes
	// in the topology have reported ready.
	TopologyStateRunning = "running"

	// TopologyStateDestroying is reported in the topology.status.topologyState field when the
	// topology is being deleted (DeletionTimestamp is set).
	TopologyStateDestroying = "destroying"

	// TopologyStateDeployFailed is reported in the topology.status.topologyState field when the
	// topology is deploying but one or more nodes have entered a terminal failure state
	// (e.g. CrashLoopBackOff or pod Failed phase).
	TopologyStateDeployFailed = "deployfailed"

	// TopologyStateDegraded is reported in the topology.status.topologyState field when the
	// topology was previously "running" but one or more nodes have since become unready or
	// entered a failure state. This distinguishes a regression from an initial startup failure.
	TopologyStateDegraded = "degraded"

	// TopologyStateDestroyFailed is reported in the topology.status.topologyState field when
	// the controller attempted to release the topology finalizer during deletion but the
	// operation failed. The object remains until the condition is resolved.
	TopologyStateDestroyFailed = "destroyfailed"

	// TopologyFinalizer is the finalizer added to Topology objects so that the controller can
	// set the TopologyState to "destroying" before the object is removed.
	TopologyFinalizer = "clabernetes/topology-state"

	// ProbeStatusPassing indicates the probe is currently passing.
	ProbeStatusPassing = "passing"

	// ProbeStatusFailing indicates the probe is currently failing.
	ProbeStatusFailing = "failing"

	// ProbeStatusUnknown indicates the probe status cannot be determined.
	ProbeStatusUnknown = "unknown"

	// ProbeStatusDisabled indicates the probe is disabled.
	ProbeStatusDisabled = "disabled"
)

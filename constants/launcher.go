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
)

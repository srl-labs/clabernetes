package constants

// Version of the clabernetes manager. Set with build flags, so leave at 0.0.0.
var Version = "0.0.0" //nolint: gochecknoglobals

const (
	// Clabernetes is the name... clabernetes.
	Clabernetes = "clabernetes"

	// Clabverter is a constant for the lovely name "clabverter".
	Clabverter = "clabverter"

	// True is a constant representing the string "true".
	True = "true"

	// False is a constant representing the string "false".
	False = "false"

	// Default is a constant for the string default -- often used for keys in clabernetes maps.
	Default = "default"

	// AppNameDefault is the default name for the "app" (the helm value appName) -- "clabernetes".
	AppNameDefault = "clabernetes"

	// VXLANServicePort is the port number for vxlan that we use in the kubernetes service.
	VXLANServicePort = 14789

	// TCP is... TCP.
	TCP = "TCP"

	// UDP is... UDP.
	UDP = "UDP"

	// LauncherDefaultImage is the default image for launchers -- this shouldn't be used normally
	// since the clabernetes chart has a default value for this.
	LauncherDefaultImage = "ghcr.io/srl-labs/clabernetes/clabernetes-launcher:latest"

	// LauncherDefaultImagePullThroughMode is the default mode for "image pull through" on the
	// launcher pods. This is "auto" the mode where we try to pull via CRI and fall back to pulling
	// via local docker daemon if we cant.
	LauncherDefaultImagePullThroughMode = "auto"

	// DefaultInClusterDNSSuffix is the default "svc.cluster.local" dns suffix.
	DefaultInClusterDNSSuffix = "svc.cluster.local"
)

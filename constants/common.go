package constants

// Version of the clabernetes manager. Set with build flags, so leave at 0.0.0.
var Version = "0.0.0" //nolint: gochecknoglobals

const (
	// Clabernetes is the name... clabernetes.
	Clabernetes = "clabernetes"

	// Clabverter is a constant for the lovely name "clabverter".
	Clabverter = "clabverter"

	// False is a constant representing the string "false".
	False = "false"

	// AppNameDefault is the default name for the "app" (the helm value appName) -- "clabernetes".
	AppNameDefault = "clabernetes"

	// VXLANPort is the port number for vxlan.
	VXLANPort = 4789

	// LauncherDefaultImage is the default image for launchers -- this shouldn't be used normally
	// since the chart has a default value for this.
	LauncherDefaultImage = "ghcr.io/carlmontanari/clabernetes/clabernetes-launcher:latest"
)

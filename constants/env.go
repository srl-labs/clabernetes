package constants

const (
	// AppNameEnvVar is the environment variable name of the "appName" as supplied to helm
	// if not set the default will always be AppNameDefault.
	AppNameEnvVar = "APP_NAME"

	// ManagerLoggerLevelEnv is the environment variable name that can be used to set the
	// clabernetes manager logger level. This is the logger for the "main" process, not the
	// individual controllers.
	ManagerLoggerLevelEnv = "MANAGER_LOGGER_LEVEL"

	// ControllerLoggerLevelEnv is the environment variable name that can be used to set the
	// clabernetes controllers logger level.
	ControllerLoggerLevelEnv = "CONTROLLER_LOGGER_LEVEL"

	// ClientOperationTimeoutMultiplierEnv is the multiplier applied to the default client
	// operation timeout.
	ClientOperationTimeoutMultiplierEnv = "CLIENT_OPERATION_TIMEOUT_MULTIPLIER"

	// LauncherLoggerLevelEnv is the environment variable name that can be used to set the
	// clabernetes launcher logger level.
	LauncherLoggerLevelEnv = "LAUNCHER_LOGGER_LEVEL"

	// LauncherImageEnv env var that tells the controllers what image to use for clabernetes
	// (launcher) pods.
	LauncherImageEnv = "LAUNCHER_IMAGE"

	// LauncherPullPolicyEnv env var that tells the controllers what pull policy to use for
	// clabernetes (launcher) pods.
	LauncherPullPolicyEnv = "LAUNCHER_PULL_POLICY"
)

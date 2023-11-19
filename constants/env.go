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

	// InClusterDNSSuffixEnv is the env var specifying the DNS suffix to use to resolve in cluster
	// services, typically 'svc.cluster.local".
	InClusterDNSSuffixEnv = "IN_CLUSTER_DNS_SUFFIX"
)

const (
	// GitHubTokenEnv is the env var that holds (optionally of course) a GitHub token -- this is
	// useful for the clabverter tool as well as the launcher where we *may* need to use the
	// GitHub api to list contents of a directory (this is specifically for dealing with large files
	// that don't fit in configmaps).
	GitHubTokenEnv = "GITHUB_TOKEN" //nolint:gosec
)

const (
	// LauncherLoggerLevelEnv is the environment variable name that can be used to set the
	// clabernetes launcher logger level.
	LauncherLoggerLevelEnv = "LAUNCHER_LOGGER_LEVEL"

	// LauncherContainerlabDebug is the environment variable name that can be used to enable the
	// debug flag of clabernetes when invoked on the launcher pod.
	LauncherContainerlabDebug = "LAUNCHER_CONTAINERLAB_DEBUG"

	// LauncherImageEnv env var that tells the controllers what image to use for clabernetes
	// (launcher) pods.
	LauncherImageEnv = "LAUNCHER_IMAGE"

	// LauncherPullPolicyEnv env var that tells the controllers what pull policy to use for
	// clabernetes (launcher) pods.
	LauncherPullPolicyEnv = "LAUNCHER_PULL_POLICY"

	// LauncherInsecureRegistries env var that tells the launcher pods which registries are
	// insecure. Should be set by the controller via the topology spec.
	LauncherInsecureRegistries = "LAUNCHER_INSECURE_REGISTRIES"

	// LauncherImagePullThroughModeEnv env var tells the manager how to configure the launcher,
	// which in turn tells the launcher how it should attempt to pull images for the node it
	// represents.
	LauncherImagePullThroughModeEnv = "LAUNCHER_IMAGE_PULL_THROUGH_MODE"

	// LauncherCRIKindEnv env var tells the launcher what CRI sock is mounted in it (if configured).
	LauncherCRIKindEnv = "LAUNCHER_CRI_KIND"

	// LauncherNodeNameEnv is the env var that holds the name of the node in the original topology
	// that a given launcher is responsible for.
	LauncherNodeNameEnv = "LAUNCHER_NODE_NAME"

	// LauncherNodeImageEnv is the env var that holds the image name of the node in the original
	// topology that a given launcher is responsible for.
	LauncherNodeImageEnv = "LAUNCHER_NODE_IMAGE"
)

const (
	// ClickerLoggerLevelEnv is the environment variable name that can be used to set the
	// cl(abernetes t)ick(l)er logger level.
	ClickerLoggerLevelEnv = "CLICKER_LOGGER_LEVEL"

	// ClickerWorkerImage is the environment variable name that can be used to set the
	// cl(abernetes t)ick(l)er worker image -- that is, the image that is deployed in a pod on all
	// target nodes, by default this is simply 'busybox'.
	ClickerWorkerImage = "CLICKER_WORKER_IMAGE"

	// ClickerWorkerCommand is the command for the worker -- defaults to "/bin/sh".
	ClickerWorkerCommand = "CLICKER_WORKER_COMMAND"

	// ClickerWorkerScript is the script for the clicker worker -- defaults to 'echo "hello, there"'
	// since we can't know what users will need here.
	ClickerWorkerScript = "CLICKER_WORKER_SCRIPT"

	// ClickerWorkerResources -- see also ClickerGlobalAnnotations -- same thing just for the worker
	// pod resources, we'll just unmarshal to k8scorev1.ResourceRequirements.
	ClickerWorkerResources = "CLICKER_WORKER_RESOURCES"

	// ClickerGlobalAnnotations is the env var where we store the global annotations from the helm
	// deployment -- these annotations need to be stored such that they can be set on the actual
	// "worker" pods as well. In "normal" clabernetes operations this is stored in the configmap
	// where other config things are stored, but in context of the clicker this configmap may not
	// exist, so we'll just stuff these into env vars.
	ClickerGlobalAnnotations = "CLICKER_GLOBAL_ANNOTATIONS"

	// ClickerGlobalLabels -- see also ClickerGlobalAnnotations -- same thing just for labels.
	ClickerGlobalLabels = "CLICKER_GLOBAL_LABELS"
)

package manager

import (
	"context"
	"math/rand"
	"time"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesmanagerprepare "github.com/srl-labs/clabernetes/manager/prepare"
	clabernetesutil "github.com/srl-labs/clabernetes/util"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

const (
	electionDuration = 60
	electionRenew    = 40
	electionRetry    = 8
)

// StartClabernetes is a function that starts the clabernetes manager.
func StartClabernetes(initializer bool) {
	if clabernetesInstance != nil {
		clabernetesutil.Panic("clabernetes instance already created...")
	}

	rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	claberneteslogging.InitManager()

	logManager := claberneteslogging.GetManager()

	clabernetesLogger := logManager.MustRegisterAndGetLogger(
		clabernetesconstants.Clabernetes,
		clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.ManagerLoggerLevelEnv,
			clabernetesconstants.Info,
		),
	)

	err := createNewKlogLogger(logManager)
	if err != nil {
		clabernetesLogger.Criticalf("failed patching klog, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	clabernetesInstance = &clabernetes{
		baseCtx: clabernetesutil.SignalHandledContext(clabernetesLogger.Criticalf),
		appName: clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.AppNameEnvVar,
			clabernetesconstants.AppNameDefault,
		),
		initializer: initializer,
		logger:      clabernetesLogger,
	}

	clabernetesInstance.startup()
}

var clabernetesInstance *clabernetes //nolint:gochecknoglobals

type clabernetes struct {
	baseCtx   context.Context
	leaderCtx context.Context

	appName string

	initializer bool

	logger claberneteslogging.Instance

	namespace  string
	kubeConfig *rest.Config
	kubeClient *kubernetes.Clientset

	mgr ctrlruntime.Manager
}

func (c *clabernetes) GetContext() context.Context {
	return c.baseCtx
}

func (c *clabernetes) GetAppName() string {
	return c.appName
}

func (c *clabernetes) GetBaseLogger() claberneteslogging.Instance {
	return c.logger
}

func (c *clabernetes) GetNamespace() string {
	return c.namespace
}

func (c *clabernetes) IsInitializer() bool {
	return c.initializer
}

func (c *clabernetes) GetKubeConfig() *rest.Config {
	return c.kubeConfig
}

func (c *clabernetes) GetKubeClient() *kubernetes.Clientset {
	return c.kubeClient
}

func (c *clabernetes) GetCtrlRuntimeMgr() ctrlruntime.Manager {
	return c.mgr
}

func (c *clabernetes) GetCtrlRuntimeClient() ctrlruntimeclient.Client {
	return c.mgr.GetClient()
}

func (c *clabernetes) NewContextWithTimeout() (context.Context, context.CancelFunc) {
	dur := clabernetesconstants.DefaultClientOperationTimeout
	mul := clabernetesutil.GetEnvFloat64OrDefault(
		clabernetesconstants.ClientOperationTimeoutMultiplierEnv,
		1,
	)

	finalDur := time.Duration(dur.Seconds()*mul) * time.Second

	c.logger.Debugf("issuing new context with timeout value '%s'", finalDur)

	if c.leaderCtx == nil {
		c.logger.Info("requesting new context but leader election context has not been set")

		return context.WithTimeout(c.baseCtx, finalDur)
	}

	return context.WithTimeout(c.leaderCtx, finalDur)
}

func (c *clabernetes) startup() {
	c.logger.Info("starting clabernetes...")

	var err error

	c.namespace, err = clabernetesutil.CurrentNamespace()
	if err != nil {
		c.logger.Criticalf("failed getting current namespace, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	c.kubeConfig, err = rest.InClusterConfig()
	if err != nil {
		c.logger.Criticalf("failed getting in cluster kubeconfig, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	c.kubeClient, err = kubernetes.NewForConfig(c.kubeConfig)
	if err != nil {
		c.logger.Criticalf("failed creating kube client from in cluster kubeconfig, err: %s", err)

		clabernetesutil.Panic(err.Error())
	}

	if c.initializer {
		// initializer means we are the init container and should run initialization tasks like
		// creating crds/webhook configs. once done with this we are done and the init process will
		// call os.exit to kill the process.
		c.startInitLeading()
	}

	clabernetesmanagerprepare.Prepare(c)

	c.startLeading()
}

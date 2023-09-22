package controllers

import (
	"context"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"

	clientgorest "k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewController defines a function that creates and returns a clabernetes Controller object.
type NewController func(
	ctx context.Context,
	appName string,
	config *clientgorest.Config,
	client ctrlruntimeclient.Client,
) Controller

// Controller defines a clabernetes controller.
type Controller interface {
	// SetupWithManager sets the given controller up with the controller-runtime manager.
	SetupWithManager(mgr ctrlruntime.Manager) error
	// Reconcile is the actual reconcile function of the controller.
	Reconcile(context.Context, ctrlruntime.Request) (ctrlruntime.Result, error)
}

// NewBaseController returns a new BaseController object to embed in clabernetes controllers.
func NewBaseController(
	ctx context.Context,
	controllerName,
	appName string,
	config *clientgorest.Config,
	client ctrlruntimeclient.Client,
) *BaseController {
	logManager := claberneteslogging.GetManager()

	logger := logManager.MustRegisterAndGetLogger(
		controllerName,
		clabernetesutil.GetEnvStrOrDefault(
			clabernetesconstants.ControllerLoggerLevelEnv,
			clabernetesconstants.Info,
		),
	)

	logger.Info("creating controller instance")

	return &BaseController{
		Ctx:                 ctx,
		AppName:             appName,
		ControllerNamespace: clabernetesutil.MustCurrentNamespace(),
		Log:                 logger,
		Config:              config,
		Client:              client,
	}
}

// BaseController is the base clabernetes controller that is embedded in all clabernetes
// controllers, it provides common attributes for the controllers such as a log instance.
type BaseController struct {
	// Ctx is the outer clabernetes context, controllers can use it to check if it is done, spawn
	// new contexts from it, and pass it to other objects such as the messaging client.
	Ctx context.Context
	// AppName is the name of the clabernetes app (the helm release name).
	AppName string
	// ControllerNamespace is the namespace the controller is running in.
	ControllerNamespace string
	Log                 claberneteslogging.Instance
	Config              *clientgorest.Config
	Client              ctrlruntimeclient.Client
}

// LogReconcileStart is a convenience/consistency function to log the start of a reconcile event.
func (c *BaseController) LogReconcileStart(req ctrlruntime.Request) {
	c.Log.Info("reconcile started")
	c.Log.Debugf("reconcile request namespace/name: %s/%s", req.Namespace, req.Name)
}

// LogReconcileStartDelete is a convenience/consistency function to log the start of a *delete*
// reconcile event.
func (c *BaseController) LogReconcileStartDelete(_ ctrlruntime.Request) {
	c.Log.Info("resource is deleting, handling deletion tasks")
}

// LogReconcileCompleteSuccess is a convenience/consistency function to log the successful
// completion of a reconcile.
func (c *BaseController) LogReconcileCompleteSuccess(_ ctrlruntime.Request) {
	c.Log.Info("reconcile completed successfully")
}

// LogReconcileCompleteObjectNotExist is a convenience/consistency function to log the successful
// completion of a reconcile when an object doesn't exist anymore.
func (c *BaseController) LogReconcileCompleteObjectNotExist(_ ctrlruntime.Request) {
	c.Log.Info("object no longer exists, reconcile completed successfully")
}

// LogReconcileFailedGettingObject is a convenience/consistency function to log an error on failure
// to get the object under reconciliation.
func (c *BaseController) LogReconcileFailedGettingObject(req ctrlruntime.Request, err error) {
	c.Log.Criticalf("failed fetching '%s/%s', error: %s", req.Namespace, req.Name, err)
}

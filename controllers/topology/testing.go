package topology

import (
	"context"

	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	clientgorest "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTestController returns a minimal topology Controller suitable for unit tests.
// It uses the provided fake client and scheme; no real Kubernetes connection is made.
func NewTestController(
	client ctrlruntimeclient.Client,
	scheme *apimachineryruntime.Scheme,
) *Controller {
	claberneteslogging.InitManager()

	logManager := claberneteslogging.GetManager()

	logger, _ := logManager.GetLogger("test")
	if logger == nil {
		logger = logManager.MustRegisterAndGetLogger(
			"test",
			clabernetesutil.GetEnvStrOrDefault("LOG_LEVEL", "disabled"),
		)
	}

	base := &clabernetescontrollers.BaseController{
		Ctx:                 context.Background(),
		AppName:             "test",
		ControllerNamespace: "default",
		Log:                 logger,
		Config:              &clientgorest.Config{},
		Client:              client,
	}

	return &Controller{
		BaseController: base,
	}
}

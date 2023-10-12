package manager

import (
	"context"
	"fmt"
	"time"

	clabernetescontrollerstopologycontainerlab "github.com/srl-labs/clabernetes/controllers/topology/containerlab"
	clabernetescontrollerstopologykne "github.com/srl-labs/clabernetes/controllers/topology/kne"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetesmanagerelection "github.com/srl-labs/clabernetes/manager/election"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	apimachineryscheme "k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func (c *clabernetes) startLeading() {
	leaderElectionIdentity := clabernetesmanagerelection.GenerateLeaderIdentity()
	leaderElectionLockName := fmt.Sprintf("%s-manager", c.appName)

	leaderElectionLock := clabernetesmanagerelection.GetLeaseLock(
		c.kubeClient,
		c.appName,
		c.namespace,
		leaderElectionLockName,
		leaderElectionIdentity,
	)

	c.logger.Info("start leader election")
	clabernetesmanagerelection.RunElection(
		c.baseCtx,
		leaderElectionIdentity,
		leaderElectionLock,
		clabernetesmanagerelection.Timers{
			Duration:      electionDuration * time.Second,
			RenewDeadline: electionRenew * time.Second,
			RetryPeriod:   electionRetry * time.Second,
		},
		c.start,
		c.stopLeading,
		c.newLeader,
	)
}

func (c *clabernetes) stopLeading() {
	c.logger.Info("stopping clabernetes...")

	clabernetesutil.Exit(clabernetesconstants.ExitCode)
}

func (c *clabernetes) newLeader(leaderElectionIdentity string) {
	c.logger.Infof("new leader elected '%s'", leaderElectionIdentity)
}

func (c *clabernetes) start(ctx context.Context) {
	c.leaderCtx = ctx

	c.logger.Debug("registering apis to scheme...")

	scheme := apimachineryruntime.NewScheme()

	apisToRegisterFuncs := []func() (apimachineryscheme.GroupVersion, []apimachineryruntime.Object){
		clabernetesapistopologyv1alpha1.GetAPIs,
	}

	for _, apiToRegisterFunc := range apisToRegisterFuncs {
		gv, objects := apiToRegisterFunc()

		for _, object := range objects {
			scheme.AddKnownTypes(gv, object)
		}

		metav1.AddToGroupVersion(scheme, gv)
	}

	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		clabernetesutil.Panic(err.Error())
	}

	err = apiextensionsv1.AddToScheme(scheme)
	if err != nil {
		clabernetesutil.Panic(err.Error())
	}

	c.logger.Debug("apis registered...")

	c.mgr = clabernetesutil.MustNewManager(scheme)

	go func() {
		err = c.mgr.Start(c.leaderCtx)
		if err != nil {
			c.logger.Criticalf(
				"encountered error starting controller-runtime manager, err: %s",
				err,
			)

			clabernetesutil.Exit(clabernetesconstants.ExitCodeError)
		}
	}()

	c.logger.Info("begin syncing controller-runtime manager cache...")

	synced := c.mgr.GetCache().WaitForCacheSync(c.leaderCtx)
	if !synced {
		c.logger.Critical("encountered error syncing controller-runtime manager cache")

		clabernetesutil.Exit(clabernetesconstants.ExitCodeError)
	}

	c.logger.Debug("controller-runtime manager cache synced...")

	c.logger.Info("registering controllers...")

	controllersToRegisterFuncs := []clabernetescontrollers.NewController{
		clabernetescontrollerstopologycontainerlab.NewController,
		clabernetescontrollerstopologykne.NewController,
	}

	for _, newF := range controllersToRegisterFuncs {
		ctrl := newF(c.baseCtx, c.appName, c.kubeConfig, c.mgr.GetClient())

		clabernetesutil.MustSetupWithManager(ctrl.SetupWithManager, c.mgr)
	}

	c.logger.Debug("controllers registered...")

	c.logger.Debug("startup complete...")

	c.logger.Info("running forever or until interrupt...")

	<-c.leaderCtx.Done()

	clabernetesutil.Exit(clabernetesconstants.ExitCode)
}

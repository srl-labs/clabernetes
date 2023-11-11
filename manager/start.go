package manager

import (
	"context"
	"fmt"
	"time"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	clabernetescontrollerstopologycontainerlab "github.com/srl-labs/clabernetes/controllers/topology/containerlab"
	clabernetescontrollerstopologykne "github.com/srl-labs/clabernetes/controllers/topology/kne"
	clabernetesmanagerelection "github.com/srl-labs/clabernetes/manager/election"
	clabernetesmanagerprestart "github.com/srl-labs/clabernetes/manager/prestart"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	"k8s.io/apimachinery/pkg/labels"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimemetricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func (c *clabernetes) startLeaderElection() {
	c.leaderElectionIdentity = clabernetesmanagerelection.GenerateLeaderIdentity()
	leaderElectionLockName := fmt.Sprintf("%s-manager", c.appName)

	leaderElectionLock := clabernetesmanagerelection.GetLeaseLock(
		c.kubeClient,
		c.appName,
		c.namespace,
		leaderElectionLockName,
		c.leaderElectionIdentity,
	)

	c.logger.Info("start leader election")

	clabernetesmanagerelection.RunElection(
		c.baseCtx,
		c.leaderElectionIdentity,
		leaderElectionLock,
		clabernetesmanagerelection.Timers{
			Duration:      electionDuration * time.Second,
			RenewDeadline: electionRenew * time.Second,
			RetryPeriod:   electionRetry * time.Second,
		},
		c.startLeading,
		c.stopLeading,
		c.newLeader,
	)
}

func (c *clabernetes) stopLeading() {
	c.logger.Info("stopping clabernetes...")

	c.Exit(clabernetesconstants.ExitCode)
}

func (c *clabernetes) newLeader(newLeaderIdentity string) {
	c.logger.Infof("new leader elected '%s'", newLeaderIdentity)

	if newLeaderIdentity != c.leaderElectionIdentity {
		c.logger.Debug(
			"new leader is not us, nothing else for us to do. setting ready state to true",
		)

		c.ready = true
	} else {
		c.logger.Debug("new leader is us, resetting ready state to false")

		c.ready = false
	}
}

func mustNewManager(scheme *apimachineryruntime.Scheme, appName string) ctrlruntime.Manager {
	mgr, err := ctrlruntime.NewManager(
		ctrlruntime.GetConfigOrDie(),
		ctrlruntime.Options{
			Logger: klog.NewKlogr(),
			Scheme: scheme,
			Metrics: ctrlruntimemetricsserver.Options{
				BindAddress: "0",
			},
			LeaderElection: false,
			NewCache: func(
				config *rest.Config,
				opts ctrlruntimecache.Options,
			) (ctrlruntimecache.Cache, error) {
				opts.DefaultLabelSelector = labels.SelectorFromSet(
					labels.Set{
						// only cache objects with the "clabernetes/app" label, why would we care
						// about anything else (for now -- and we can override it with opts.ByObject
						// anyway?! and... who the hell calls their app "clabernetes" so this should
						// really limit the cache nicely :)
						"clabernetes/app": appName,
					},
				)

				// obviously we need to cache all "our" objects, so do that
				opts.ByObject = map[ctrlruntimeclient.Object]ctrlruntimecache.ByObject{
					&clabernetesapistopologyv1alpha1.Containerlab{}: {
						Namespaces: map[string]ctrlruntimecache.Config{
							ctrlruntimecache.AllNamespaces: {
								LabelSelector: labels.Everything(),
							},
						},
					},
					&clabernetesapistopologyv1alpha1.Kne{}: {
						Namespaces: map[string]ctrlruntimecache.Config{
							ctrlruntimecache.AllNamespaces: {
								LabelSelector: labels.Everything(),
							},
						},
					},
				}

				return ctrlruntimecache.New(config, opts)
			},
		},
	)
	if err != nil {
		clabernetesutil.Panic(fmt.Sprintf("unable to start manager, error: %s", err))
	}

	return mgr
}

func (c *clabernetes) startLeading(ctx context.Context) {
	c.leaderCtx = ctx

	c.logger.Info("begin pre-start...")

	clabernetesmanagerprestart.PreStart(c)

	c.logger.Debug("pre-start complete...")

	go func() {
		err := c.mgr.Start(c.leaderCtx)
		if err != nil {
			c.logger.Criticalf(
				"encountered error starting controller-runtime manager, err: %s",
				err,
			)

			c.Exit(clabernetesconstants.ExitCodeError)
		}
	}()

	c.logger.Info("begin syncing controller-runtime manager cache...")

	synced := c.mgr.GetCache().WaitForCacheSync(c.leaderCtx)
	if !synced {
		c.logger.Critical("encountered error syncing controller-runtime manager cache")

		c.Exit(clabernetesconstants.ExitCodeError)
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

	c.ready = true

	c.logger.Debug("startup complete...")

	c.logger.Info("running forever or until interrupt...")

	<-c.leaderCtx.Done()
}

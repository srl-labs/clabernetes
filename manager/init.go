package manager

import (
	"context"
	"fmt"
	"os"
	"time"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	clabernetesmanagerelection "gitlab.com/carlmontanari/clabernetes/manager/election"
	clabernetesmanagerinitialize "gitlab.com/carlmontanari/clabernetes/manager/initialize"
)

func (c *clabernetes) startInitLeading() {
	leaderElectionIdentity := clabernetesmanagerelection.GenerateLeaderIdentity()
	leaderElectionLockName := fmt.Sprintf("%s-init", c.appName)

	leaderElectionLock := clabernetesmanagerelection.GetLeaseLock(
		c.kubeClient,
		c.appName,
		c.namespace,
		leaderElectionLockName,
		leaderElectionIdentity,
	)

	c.logger.Info("start init leader election")
	clabernetesmanagerelection.RunElection(
		c.baseCtx,
		leaderElectionIdentity,
		leaderElectionLock,
		clabernetesmanagerelection.Timers{
			Duration:      electionDuration * time.Second,
			RenewDeadline: electionRenew * time.Second,
			RetryPeriod:   electionRetry * time.Second,
		},
		c.init,
		c.stopLeading,
		c.newLeader,
	)
}

func (c *clabernetes) init(ctx context.Context) {
	c.logger.Info("begin init...")

	c.leaderCtx = ctx

	clabernetesmanagerinitialize.Initialize(c)

	c.logger.Info("init complete...")

	os.Exit(clabernetesconstants.ExitCode)
}

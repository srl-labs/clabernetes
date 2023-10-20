package election

import (
	"context"

	"k8s.io/client-go/tools/leaderelection"
)

// RunElection runs the leader election process executing the provided callbacks for
// OnStartedLeading and OnStoppedLeading when those events happen.
func RunElection(
	ctx context.Context,
	leaderElectionIdentity string,
	lock *ClabernetesLeaseLock,
	timers Timers,
	startLeading func(c context.Context),
	stopLeading func(),
	newLeader func(newLeaderIdentity string),
) {
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Name:            leaderElectionIdentity,
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   timers.Duration,
		RenewDeadline:   timers.RenewDeadline,
		RetryPeriod:     timers.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(c context.Context) {
				startLeading(c)
			},
			OnStoppedLeading: func() {
				stopLeading()
			},
			OnNewLeader: func(newLeaderIdentity string) {
				newLeader(newLeaderIdentity)
			},
		},
	})
}

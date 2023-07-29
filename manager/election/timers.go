package election

import "time"

// Timers holds duration values for leader election/lease.
type Timers struct {
	// Duration of the leader election lease.
	Duration time.Duration
	// RenewDeadline for renewing leader election lease.
	RenewDeadline time.Duration
	// RetryPeriod window for leader election lease retries.
	RetryPeriod time.Duration
}

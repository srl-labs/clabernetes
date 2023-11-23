package constants

import "time"

const (
	// DefaultClientOperationTimeout is the default timeout to use for all kubernetes client
	// operations (i.e. Get, Create, etc.).
	DefaultClientOperationTimeout = 2 * time.Second

	// PullerPodTimeout is the max time to wait for puller pod spawning in both the http server
	// where we handle puller pod requests and in the launcher when we wait for the image to be
	// available.
	PullerPodTimeout = 5 * time.Minute
)

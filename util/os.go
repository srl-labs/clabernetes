package util

import (
	"os"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

// Exit flushes the logging manager and exits the program w/ the given exit code.
func Exit(exitCode int) {
	claberneteslogging.GetManager().Flush()

	os.Exit(exitCode)
}

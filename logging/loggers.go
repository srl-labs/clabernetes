package logging

import (
	"fmt"
	"os"
)

// the default and very simple logger.
func printLog(a ...any) {
	fmt.Println(a...) //nolint:forbidigo
}

// StdErrLog writes `a` to stderr, it ignores failures.
func StdErrLog(a ...any) {
	_, _ = fmt.Fprintln(os.Stderr, a...)
}

package testhelper

import (
	"fmt"
	"os/exec"
	"testing"
)

// YQCommand accepts some yaml content and returns it after executing the given yqPattern against
// it.
func YQCommand(t *testing.T, content []byte, yqPattern string) []byte {
	t.Helper()

	yqCmd := fmt.Sprintf("echo '%s' | yq '%s'", string(content), yqPattern)

	cmd := exec.Command( //nolint:gosec
		"bash",
		"-c",
		yqCmd,
	)

	return Execute(t, cmd)
}

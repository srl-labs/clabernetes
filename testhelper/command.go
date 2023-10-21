package testhelper

import (
	"os/exec"
	"testing"
)

// Execute executes a command in the context of a test.
func Execute(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("error executing command, error: %q", err)
		t.Logf("errored command: %s", cmd.String())
		t.Logf("errored command combined output: %s", output)
		t.FailNow()
	}

	return output
}

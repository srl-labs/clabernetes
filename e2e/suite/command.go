package suite

import (
	"fmt"
	"os/exec"
	"testing"
)

const (
	kubectl = "kubectl"
)

func execute(t *testing.T, cmd *exec.Cmd) []byte {
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

func kubectlNamespace(t *testing.T, operation Operation, namespace string) {
	t.Helper()

	cmd := exec.Command(kubectl, string(operation), "namespace", namespace) //nolint:gosec

	err := cmd.Run()
	if err != nil {
		t.Fatalf("error executing kubectl command, error: '%s'", err)
	}
}

// KubectlCreateNamespace execs a kubectl create namespace.
func KubectlCreateNamespace(t *testing.T, namespace string) {
	t.Helper()

	kubectlNamespace(t, Create, namespace)
}

// KubectlDeleteNamespace execs a kubectl delete namespace.
func KubectlDeleteNamespace(t *testing.T, namespace string) {
	t.Helper()

	kubectlNamespace(t, Delete, namespace)
}

// KubectlFileOp execs a kubectl operation on a file (i.e. apply/delete).
func KubectlFileOp(t *testing.T, operation Operation, namespace, fileName string) {
	t.Helper()

	cmd := exec.Command( //nolint:gosec
		kubectl,
		string(operation),
		"--namespace",
		namespace,
		"-f",
		fileName,
	)

	_ = execute(t, cmd)
}

// KubectlGetOp runs get on the given object, returning the yaml output.
func KubectlGetOp(t *testing.T, kind, namespace, name string) []byte {
	t.Helper()

	cmd := exec.Command( //nolint:gosec
		kubectl,
		string(Get),
		kind,
		"--namespace",
		namespace,
		name,
		"-o",
		"yaml",
	)

	return execute(t, cmd)
}

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

	return execute(t, cmd)
}

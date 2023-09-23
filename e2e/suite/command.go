package suite

import (
	"fmt"
	"os/exec"
	"testing"
)

const (
	kubectl = "kubectl"
)

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

	err := cmd.Run()
	if err != nil {
		t.Fatalf("error executing kubectl command, error: '%s'", err)
	}
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

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("error executing kubectl command, error: '%s'", err)
	}

	return output
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

	output, err := cmd.Output()
	if err != nil {
		t.Log("yq command:", yqCmd)
		t.Fatalf("error executing yq command, error: '%s'", err)
	}

	return output
}

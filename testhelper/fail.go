package testhelper

import (
	"encoding/json"
	"os"
	"testing"

	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
)

// FailOutput is a simple func to nicely print actual vs expected output when a test fails.
func FailOutput(t *testing.T, actual, expected interface{}) {
	t.Helper()

	switch actual.(type) {
	case string, []byte:
		t.Fatalf(
			"actual and expected outputs do not match..."+
				"\n\033[0;36m*** actual   >>>\033[0m"+
				"\n%s"+
				"\n\033[0;36m<<< actual   ***\033[0m"+
				"\n\033[0;35m*** expected >>>\033[0m"+
				"\n%s"+
				"\n\033[0;35m<<< expected ***\033[0m",
			actual,
			expected,
		)
	default:
		t.Fatalf(
			"actual and expected outputs do not match..."+
				"\n\033[0;36m*** actual   >>>\033[0m"+
				"\n%+v"+
				"\n\033[0;36m<<< actual   ***\033[0m"+
				"\n\033[0;35m*** expected >>>\033[0m"+
				"\n%+v"+
				"\n\033[0;35m<<< expected ***\033[0m",
			actual,
			expected,
		)
	}
}

// ReadFile is a helper to read a (usually) golden file.
func ReadFile(t *testing.T, f string) []byte {
	t.Helper()

	content, err := os.ReadFile(f) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}

	return content
}

// WriteJSON is a helper to write json to the specified file.
func WriteJSON(t *testing.T, f string, o interface{}) {
	t.Helper()

	j, err := json.MarshalIndent(o, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(f, j, clabernetesconstants.PermissionsEveryoneReadUserWrite)
	if err != nil {
		t.Fatal(err)
	}
}

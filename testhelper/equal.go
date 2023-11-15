package testhelper

import (
	"bytes"
	"encoding/json"
	"testing"
)

// MarshaledEqual marshals (json marshal) a and b and bytes.Equal compares them. If they are not
// equal FailOutput is called.
func MarshaledEqual(t *testing.T, a, b any) {
	t.Helper()

	aJSON, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		t.Fatalf(
			"failed marshaling actual, err: %s",
			err,
		)
	}

	bJSON, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		t.Fatalf(
			"failed marshaling expected, err: %s",
			err,
		)
	}

	if !bytes.Equal(aJSON, bJSON) {
		FailOutput(t, aJSON, bJSON)
	}
}

package suite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// GlobStepFixtures globs all fixtures for the given test step index and returns the filenames.
func GlobStepFixtures(t *testing.T, stepIndex int) []string {
	t.Helper()

	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed getting test directory, error: '%s'", err)
	}

	files, err := filepath.Glob(fmt.Sprintf("%s/test-fixtures/%d-*.yaml", testDir, stepIndex))
	if err != nil {
		t.Fatalf("failed globbing test fixtures, error: '%s'", err)
	}

	return files
}

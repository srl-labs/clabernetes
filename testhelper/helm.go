package testhelper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
)

const (
	helm = "helm"
)

// HelmTest executes a test against a helm chart -- this is a very simple/dumb test meant only to
// ensure that we don't accidentally screw up charts. We do this by storing the "golden" output of
// a rendered chart (and subcharts if applicable) with a given values file.
func HelmTest(t *testing.T, testName, valuesFileName string) {
	t.Helper()

	// we have to make the chartname/templates dir too since thats where helm wants to write things
	actualRootDir := fmt.Sprintf("test-fixtures/%s-actual", testName)
	actualDir := fmt.Sprintf("%s/clabernetes/templates", actualRootDir)

	var valuesFile string

	if valuesFileName != "" {
		var err error

		valuesFile, err = filepath.Abs(fmt.Sprintf("test-fixtures/%s-values.yaml", testName))
		if err != nil {
			t.Fatalf(
				"failed getting abspath for values file, error: %s", err,
			)
		}
	}

	err := os.MkdirAll(actualDir, clabernetesconstants.PermissionsEveryoneRead)
	if err != nil {
		t.Fatalf(
			"failed creating actual output directory %q, error: %s", actualDir, err,
		)
	}

	defer func() {
		if !*SkipCleanup {
			err = os.RemoveAll(actualRootDir)
			if err != nil {
				t.Logf("failed cleaning up actual output directory %q, error: %s", actualDir, err)
			}
		}
	}()

	args := []string{
		"template",
		"../../.",
		"--output-dir",
		actualRootDir,
	}

	if valuesFile != "" {
		args = append(args, "--values", valuesFile)
	}

	HelmCommand(
		t,
		args...,
	)

	renderedTemplates := ReadAllRenderedTemplates(t, actualRootDir)

	if *Update {
		for expectedFileName, expectedFileContent := range renderedTemplates {
			WriteTestFixtureFile(
				t,
				fmt.Sprintf("golden/%s", filepath.Base(expectedFileName)),
				expectedFileContent,
			)
		}

		// we just wrote the golden file of course it will match, no need to check
		return
	}

	for expectedFileName, actualContents := range renderedTemplates {
		expected := ReadTestFixtureFile(t, fmt.Sprintf("golden/%s", expectedFileName))

		if !bytes.Equal(
			actualContents,
			expected,
		) {
			FailOutput(t, actualContents, expected)
		}
	}
}

// HelmCommand executes helm with the given arguments.
func HelmCommand(t *testing.T, args ...string) []byte {
	t.Helper()

	cmd := exec.Command(
		helm,
		args...,
	)

	return Execute(t, cmd)
}

// ReadAllRenderedTemplates loads all rendered template content into a map -- sub-charts are loaded
// with a "_subchart-<CHARTNAME>-" prefix.
func ReadAllRenderedTemplates(t *testing.T, rootRenderDir string) map[string][]byte {
	t.Helper()

	renderedTemplates := map[string][]byte{}

	parentChartFileNames, err := filepath.Glob(fmt.Sprintf("%s/*/templates/*.yaml", rootRenderDir))
	if err != nil {
		t.Fatalf("failed globbing parent chart files, error: '%s'", err)
	}

	subChartFileNames, err := filepath.Glob(
		fmt.Sprintf("%s/*/charts/*/templates/*.yaml", rootRenderDir),
	)
	if err != nil {
		t.Fatalf("failed globbing dependency chart files, error: '%s'", err)
	}

	for _, parentChartFileName := range parentChartFileNames {
		var contents []byte

		contents, err = os.ReadFile(parentChartFileName) //nolint:gosec
		if err != nil {
			t.Fatalf(
				"failed reading contents of actual output file %q, error: %s",
				parentChartFileName,
				err,
			)
		}

		renderedTemplates[filepath.Base(parentChartFileName)] = contents
	}

	for _, subChartFileName := range subChartFileNames {
		subChartPathComponents := strings.Split(subChartFileName, string(filepath.Separator))

		subChartName := subChartPathComponents[4]

		var contents []byte

		contents, err = os.ReadFile(subChartFileName) //nolint:gosec
		if err != nil {
			t.Fatalf(
				"failed reading contents of actual output file %q, error: %s",
				subChartFileName,
				err,
			)
		}

		renderedTemplates[fmt.Sprintf("_subchart-%s-%s", subChartName, filepath.Base(subChartFileName))] = contents //nolint:lll
	}

	return renderedTemplates
}

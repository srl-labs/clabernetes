package default_values_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
)

func TestMain(m *testing.M) {
	clabernetestesthelper.Flags()

	os.Exit(m.Run())
}

// TestDefaultValues -- really just here to ensure that we dont accidentally break our charts; this
// will probably be *highly* irritating in times of lots of chart updates, but, once we know the
// template are in a good place we can always just re-generate the "golden" outputs.
func TestDefaultValues(t *testing.T) {
	t.Parallel()

	testName := "default-values"

	// we have to make the chartname/templates dir too since thats where helm wants to write things
	actualRootDir := fmt.Sprintf("test-fixtures/%s-actual", testName)
	actualDir := fmt.Sprintf("%s/clicker/templates", actualRootDir)

	err := os.MkdirAll(actualDir, clabernetesconstants.PermissionsEveryoneReadWriteOwnerExecute)
	if err != nil {
		t.Fatalf(
			"failed creating actual output directory %q, error: %s", actualDir, err,
		)
	}

	defer func() {
		if !*clabernetestesthelper.SkipCleanup {
			err = os.RemoveAll(actualRootDir)
			if err != nil {
				t.Logf("failed cleaning up actual output directory %q, error: %s", actualDir, err)
			}
		}
	}()

	clabernetestesthelper.HelmCommand(
		t,
		"template",
		"../../.",
		"--output-dir",
		actualRootDir,
	)

	var actualFileNames []string

	actualFileNames, err = filepath.Glob(fmt.Sprintf("%s/*.yaml", actualDir))
	if err != nil {
		t.Fatalf("failed globbing actual files, error: '%s'", err)
	}

	actualFileContents := map[string][]byte{}

	for _, actualFileName := range actualFileNames {
		var actualFileContent []byte

		actualFileContent, err = os.ReadFile(actualFileName) //nolint:gosec
		if err != nil {
			t.Fatalf(
				"failed reading contents of actual output file %q, error: %s", actualFileName, err,
			)
		}

		actualFileContents[actualFileName] = actualFileContent
	}

	if *clabernetestesthelper.Update {
		for actualFileName, actualFileContent := range actualFileContents {
			clabernetestesthelper.WriteTestFixtureFile(
				t,
				fmt.Sprintf("golden/%s", filepath.Base(actualFileName)),
				actualFileContent,
			)
		}

		// we just wrote the golden file of course it will match, no need to check
		return
	}
}

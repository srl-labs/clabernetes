package default_values_test

import (
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

	testName := "default_values"
	chartName := "clabernetes"

	chartsDir, err := filepath.Abs("../../..")
	if err != nil {
		t.Error(err)
	}

	clabernetestesthelper.HelmTest(
		t,
		chartName,
		testName,
		clabernetesconstants.Clabernetes,
		"",
		chartsDir,
	)
}

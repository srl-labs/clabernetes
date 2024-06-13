package default_values_test

import (
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	"os"
	"path/filepath"
	"testing"

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
	chartName := "clicker"

	chartsDir, err := filepath.Abs("../../..")
	if err != nil {
		t.Error(err)
	}

	t.Log(chartsDir)

	clabernetestesthelper.HelmTest(
		t,
		chartName,
		testName,
		clabernetesconstants.Clabernetes,
		"",
		chartsDir,
	)
}

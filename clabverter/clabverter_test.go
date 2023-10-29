package clabverter_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	clabernetesclabverter "github.com/srl-labs/clabernetes/clabverter"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
)

func TestClabvert(t *testing.T) {
	cases := []struct {
		name                 string
		topologyFile         string
		destinationNamespace string
		insecureRegistries   string
	}{
		{
			name:                 "simple",
			topologyFile:         "test-fixtures/clabversiontest/clab.yaml",
			destinationNamespace: "notclabernetes",
			insecureRegistries:   "1.2.3.4",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actualDir := fmt.Sprintf("test-fixtures/%s-actual", testCase.name)

				err := os.MkdirAll(actualDir, clabernetesconstants.PermissionsEveryoneRead)
				if err != nil {
					t.Fatalf(
						"failed creating actual output directory %q, error: %s", actualDir, err,
					)
				}

				defer func() {
					if !*clabernetestesthelper.SkipCleanup {
						err = os.RemoveAll(actualDir)
						if err != nil {
							t.Logf(
								"failed cleaning up actual output directory %q, error: %s",
								actualDir,
								err,
							)
						}
					}
				}()

				clabverter := clabernetesclabverter.MustNewClabverter(
					testCase.topologyFile,
					actualDir,
					testCase.destinationNamespace,
					testCase.insecureRegistries,
					false,
					true,
					false,
				)

				err = clabverter.Clabvert()
				if err != nil {
					t.Fatalf("error running clabvert, err: %s", err)
				}

				renderedTemplates := readAllManifests(t, actualDir)

				if *clabernetestesthelper.Update {
					for expectedFileName, expectedFileContent := range renderedTemplates {
						clabernetestesthelper.WriteTestFixtureFile(
							t,
							fmt.Sprintf("golden/%s", filepath.Base(expectedFileName)),
							expectedFileContent,
						)
					}

					// we just wrote the golden file of course it will match, no need to check
					return
				}

				for expectedFileName, actualContents := range renderedTemplates {
					expected := clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf("golden/%s", expectedFileName),
					)

					if !bytes.Equal(
						actualContents,
						expected,
					) {
						clabernetestesthelper.FailOutput(t, actualContents, expected)
					}
				}
			})
	}
}

func readAllManifests(t *testing.T, actualDir string) map[string][]byte {
	t.Helper()

	manifests := map[string][]byte{}

	manifestFileNames, err := filepath.Glob(fmt.Sprintf("%s/*.yaml", actualDir))
	if err != nil {
		t.Fatalf("failed globbing parent chart files, error: '%s'", err)
	}

	for _, manifestFileName := range manifestFileNames {
		var contents []byte

		contents, err = os.ReadFile(manifestFileName) //nolint:gosec
		if err != nil {
			t.Fatalf(
				"failed reading contents of actual output file %q, err: %s",
				manifestFileName,
				err,
			)
		}

		manifests[filepath.Base(manifestFileName)] = contents
	}

	return manifests
}

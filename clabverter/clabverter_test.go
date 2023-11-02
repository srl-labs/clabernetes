package clabverter_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
						expectedFileContent = normalizeManifest(t, expectedFileContent)

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

					actualContents = normalizeManifest(t, actualContents)

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

func normalizeManifest(t *testing.T, b []byte) []byte {
	t.Helper()

	switch {
	case bytes.Contains(b, []byte("kind: ConfigMap")):
		return normalizeConfigMapPaths(t, b)
	case bytes.Contains(b, []byte("kind: Containerlab")):
		return normalizeFromFileFilePaths(t, b)
	default:
		return b
	}
}

func normalizeFromFileFilePaths(t *testing.T, b []byte) []byte {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed getting working dir, err: %s", err)
	}

	originalPathPattern := regexp.MustCompile(fmt.Sprintf(`(?:filePath: )(%s)`, cwd))

	foundPathSubMatch := originalPathPattern.FindSubmatch(b)
	if len(foundPathSubMatch) != 2 {
		return b
	}

	foundPath := string(foundPathSubMatch[1])

	pathPattern := regexp.MustCompile(fmt.Sprintf(`filePath: %s`, foundPath))

	b = pathPattern.ReplaceAll(b, []byte("filePath: /some/dir/clabernetes/clabverter"))

	// above is just replacing the filePath parts, below we just pave over configmap paths because
	// its not worth the effort to try to ensure that they are the same since they can change based
	// on path of where the test is ran and then the safe concat name hash comes into play etc

	configMapPathsPattern := regexp.MustCompile(`(?m)^\s+configMapPath: .*$`)

	b = configMapPathsPattern.ReplaceAll(b, []byte("          configMapPath: REPLACED"))

	return b
}

func normalizeConfigMapPaths(t *testing.T, b []byte) []byte {
	t.Helper()

	// see also normalize file paths, not worth fighting with paths and hashes
	pathPattern := regexp.MustCompile(`(?m)^ {2}.*?: \|-$`)

	b = pathPattern.ReplaceAll(b, []byte("  REPLACED: |-"))

	return b
}

package basic_test

import (
	"fmt"
	"os"
	"testing"

	clabernetese2esuite "github.com/srl-labs/clabernetes/e2e/suite"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
)

func TestMain(m *testing.M) {
	clabernetestesthelper.Flags()

	os.Exit(m.Run())
}

func normalizeExposeService(t *testing.T, objectData []byte) []byte {
	t.Helper()

	// cluster ips obviously are going to be different all the time so we'll ignore them
	objectData = clabernetese2esuite.YQCommand(t, objectData, "del(.spec.clusterIP)")
	objectData = clabernetese2esuite.YQCommand(t, objectData, "del(.spec.clusterIPs)")

	// remove node ports since they'll be random
	objectData = clabernetese2esuite.YQCommand(t, objectData, "del(.spec.ports[].nodePort)")

	return objectData
}

func TestContainerlabBasic(t *testing.T) {
	t.Parallel()

	testName := "containerlab-basic"

	steps := clabernetese2esuite.Steps{
		{
			// this step, while obviously very "basic" does quite a bit of work for us... it ensures
			// that the default ports are allocated, the config is hashed and subdivided up, and our
			// defaults are set properly. more than that, this also asserts that the service(s) are
			// setup as we'd expect.
			Index:       10,
			Description: "Create a simple containerlab topology with just one node",
			AssertObjects: map[string][]clabernetese2esuite.AssertObject{
				"containerlab": {
					{
						Name: testName,
					},
				},
				"service": {
					{
						Name: fmt.Sprintf("%s-srl1", testName),
						NormalizeFuncs: []func(t *testing.T, objectData []byte) []byte{
							normalizeExposeService,
						},
					},
				},
			},
		},
	}

	clabernetese2esuite.Run(t, steps, testName)
}

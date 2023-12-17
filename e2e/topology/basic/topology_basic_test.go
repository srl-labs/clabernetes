package basic_test

import (
	"fmt"
	"os"
	"testing"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetestesthelpersuite "github.com/srl-labs/clabernetes/testhelper/suite"
)

func TestMain(m *testing.M) {
	clabernetestesthelper.Flags()

	os.Exit(m.Run())
}

func TestContainerlabBasic(t *testing.T) {
	t.Parallel()

	testName := "topology-basic"

	namespace := clabernetestesthelper.NewTestNamespace(testName)

	steps := clabernetestesthelpersuite.Steps{
		{
			// this step, while obviously very "basic" does quite a bit of work for us... it ensures
			// that the default ports are allocated, the config is hashed and subdivided up, and our
			// defaults are set properly. more than that, this also asserts that the service(s) are
			// setup as we'd expect.
			Index:       10,
			Description: "Create a simple containerlab topology with just one node",
			AssertObjects: map[string][]clabernetestesthelpersuite.AssertObject{
				"topology": {
					{
						Name: testName,
						NormalizeFuncs: []func(t *testing.T, objectData []byte) []byte{
							clabernetestesthelper.NormalizeTopology,
						},
					},
				},
				"service": {
					{
						Name: fmt.Sprintf("%s-srl1", testName),
						NormalizeFuncs: []func(t *testing.T, objectData []byte) []byte{
							clabernetestesthelper.NormalizeExposeService,
						},
					},
				},
			},
		},
	}

	clabernetestesthelpersuite.Run(t, steps, namespace)
}

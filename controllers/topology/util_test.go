package topology_test

import (
	"testing"

	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
)

func TestGetTopologyKind(t *testing.T) {
	cases := []struct {
		name     string
		in       *clabernetesapisv1alpha1.Topology
		expected string
	}{
		{
			name:     "unset-default-to-containerlab",
			in:       &clabernetesapisv1alpha1.Topology{},
			expected: clabernetesapis.TopologyKindContainerlab,
		},
		{
			name: "containerlab",
			in: &clabernetesapisv1alpha1.Topology{
				Spec: clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: "something",
					},
				},
			},
			expected: clabernetesapis.TopologyKindContainerlab,
		},
		{
			name: "kne",
			in: &clabernetesapisv1alpha1.Topology{
				Spec: clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Kne: "something",
					},
				},
			},
			expected: clabernetesapis.TopologyKindKne,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetescontrollerstopology.GetTopologyKind(testCase.in)
				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

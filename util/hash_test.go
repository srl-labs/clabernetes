package util_test

import (
	"testing"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func TestHashBytes(t *testing.T) {
	cases := []struct {
		name     string
		in       []byte
		expected string
	}{
		{
			name:     "simple",
			in:       []byte("hashmeplz"),
			expected: "b370f837831aa6b19d65cac4a0f8ef13e8d145027d3b95992232ccb8f0e564b5",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutil.HashBytes(testCase.in)
				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

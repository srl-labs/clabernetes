package kubernetes_test

import (
	"testing"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
)

func TestSafeConcatNameKubernetes(t *testing.T) {
	cases := []struct {
		name     string
		in       []string
		expected string
	}{
		{
			name:     "simple",
			in:       []string{"afinename"},
			expected: "afinename",
		},
		{
			name:     "simple-multi-word",
			in:       []string{"a", "fine", "name"},
			expected: "a-fine-name",
		},
		{
			name: "over-max-len",
			in: []string{
				"a",
				"fine",
				"name",
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			expected: "a-fine-name-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-8fa96d7",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutilkubernetes.SafeConcatNameKubernetes(testCase.in...)
				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

func TestSafeConcatNameMax(t *testing.T) {
	cases := []struct {
		name     string
		in       []string
		max      int
		expected string
	}{
		{
			name:     "simple",
			in:       []string{"afinename"},
			max:      30,
			expected: "afinename",
		},
		{
			name:     "simple-multi-word",
			in:       []string{"a", "fine", "name"},
			max:      30,
			expected: "a-fine-name",
		},
		{
			name: "over-max-len",
			in: []string{
				"a",
				"fine",
				"name",
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			max:      30,
			expected: "a-fine-name-xxxxxxxxxx-8fa96d7",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutilkubernetes.SafeConcatNameMax(testCase.in, testCase.max)
				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

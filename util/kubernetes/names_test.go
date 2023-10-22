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
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", //nolint:lll
			},
			expected: "a-fine-name-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-8fa96d7",
		},
	}

	for _, tc := range cases {
		t.Logf("%s: starting", tc.name)

		actual := clabernetesutilkubernetes.SafeConcatNameKubernetes(tc.in...)
		if actual != tc.expected {
			clabernetestesthelper.FailOutput(t, actual, tc.expected)
		}
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
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", //nolint:lll
			},
			max:      30,
			expected: "a-fine-name-xxxxxxxxxx-8fa96d7",
		},
	}

	for _, tc := range cases {
		t.Logf("%s: starting", tc.name)

		actual := clabernetesutilkubernetes.SafeConcatNameMax(tc.in, tc.max)
		if actual != tc.expected {
			clabernetestesthelper.FailOutput(t, actual, tc.expected)
		}
	}
}

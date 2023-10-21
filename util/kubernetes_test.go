package util_test

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	k8scorev1 "k8s.io/api/core/v1"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
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

		actual := clabernetesutil.SafeConcatNameKubernetes(tc.in...)
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

		actual := clabernetesutil.SafeConcatNameMax(tc.in, tc.max)
		if actual != tc.expected {
			clabernetestesthelper.FailOutput(t, actual, tc.expected)
		}
	}
}

func TestYAMLToK8sResourceRequirements(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected *k8scorev1.ResourceRequirements
	}{
		{
			name: "simple",
			in: `---
requests:
  memory: 128Mi
  cpu: 50m
`,
			expected: &k8scorev1.ResourceRequirements{
				Limits: k8scorev1.ResourceList{},
				Requests: k8scorev1.ResourceList{
					"memory": resource.MustParse("128Mi"),
					"cpu":    resource.MustParse("50m"),
				},
			},
		},
		{
			name: "simple",
			in: `---
requests:
  memory: 128Mi
  cpu: 50m
limits:
  memory: 256Mi
  cpu: 100m
`,
			expected: &k8scorev1.ResourceRequirements{
				Limits: k8scorev1.ResourceList{
					"memory": resource.MustParse("256Mi"),
					"cpu":    resource.MustParse("100m"),
				},
				Requests: k8scorev1.ResourceList{
					"memory": resource.MustParse("128Mi"),
					"cpu":    resource.MustParse("50m"),
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual, err := clabernetesutil.YAMLToK8sResourceRequirements(testCase.in)
				if err != nil {
					t.Fatalf(
						"failed calling YAMLToK8sResourceRequirements, error: %s", err,
					)
				}

				if !reflect.DeepEqual(actual, testCase.expected) {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

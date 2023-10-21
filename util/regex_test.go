package util_test

import (
	"regexp"
	"testing"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func TestRegexStringSubmatchToMap(t *testing.T) {
	cases := []struct {
		name     string
		p        *regexp.Regexp
		in       string
		expected map[string]string
	}{
		{
			name: "bool",
			p: regexp.MustCompile(
				`(?mi)https?:\/\/(?:www\.)?github\.com\/(?P<GroupRepo>.*?\/.*?)\/(?:(blob)|(tree))(?P<Path>.*)`, //nolint:lll
			),
			in: "https://github.com/srl-labs/containerlab/tree/main/lab-examples/srl02",
			expected: map[string]string{
				"GroupRepo": "srl-labs/containerlab",
				"Path":      "/main/lab-examples/srl02",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutil.RegexStringSubMatchToMap(testCase.p, testCase.in)

				for k, v := range testCase.expected {
					expectedV, ok := actual[k]
					if !ok || expectedV != v {
						clabernetestesthelper.FailOutput(t, actual, testCase.expected)
					}
				}
			})
	}
}

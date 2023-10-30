package util_test

import (
	"testing"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

func TestIsURL(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected bool
	}{
		{
			name:     "http",
			in:       "http://blah.com",
			expected: true,
		},
		{
			name:     "https",
			in:       "https://blah.com",
			expected: true,
		},
		{
			name:     "file",
			in:       "/some/file",
			expected: false,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutil.IsURL(testCase.in)

				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

func TestGitHubNormalLinkToRawLink(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "not-a-github-link",
			in:       "http://blah.com",
			expected: "http://blah.com",
		},
		{
			name:     "github-normal-link",
			in:       "https://github.com/srl-labs/srl-telemetry-lab/blob/main/configs/grafana/dashboards.yml",
			expected: "https://raw.githubusercontent.com/srl-labs/srl-telemetry-lab//main/configs/grafana/dashboards.yml",
		},
		{
			name:     "gitub-already-raw-link",
			in:       "https://raw.githubusercontent.com/srl-labs/srl-telemetry-lab/main/configs/grafana/dashboards.yml",
			expected: "https://raw.githubusercontent.com/srl-labs/srl-telemetry-lab/main/configs/grafana/dashboards.yml",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				actual := clabernetesutil.GitHubNormalToRawLink(testCase.in)
				if actual != testCase.expected {
					clabernetestesthelper.FailOutput(t, actual, testCase.expected)
				}
			})
	}
}

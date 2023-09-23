package testhelper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/carlmontanari/difflibgo/difflibgo"
)

// FailOutput is a simple func to nicely print actual vs expected output when a test fails.
func FailOutput(t *testing.T, actual, expected any) {
	t.Helper()

	switch actual.(type) {
	case string, []byte:
		diff := unifiedDiff(t, actual, expected)

		actualExpectedOut := fmt.Sprintf("\n\033[0;36m*** actual   >>>\033[0m"+
			"\n%s"+
			"\n\033[0;36m<<< actual   ***\033[0m"+
			"\n\033[0;35m*** expected >>>\033[0m"+
			"\n%s"+
			"\n\033[0;35m<<< expected ***\033[0m",
			actual, expected,
		)
		diffOut := fmt.Sprintf("\n\033[0;97m*** diff     >>>\033[0m"+
			"\n%s"+
			"\n\033[0;97m<<< diff     ***\033[0m", diff)

		if *OnlyDiff {
			t.Fatalf(
				"actual and expected outputs do not match...%s",
				diffOut,
			)
		} else {
			t.Fatalf(
				"actual and expected outputs do not match...%s%s",
				actualExpectedOut,
				diffOut,
			)
		}
	default:
		t.Fatalf(
			"actual and expected outputs do not match..."+
				"\n\033[0;36m*** actual   >>>\033[0m"+
				"\n%+v"+
				"\n\033[0;36m<<< actual   ***\033[0m"+
				"\n\033[0;35m*** expected >>>\033[0m"+
				"\n%+v"+
				"\n\033[0;35m<<< expected ***\033[0m",
			actual,
			expected,
		)
	}
}

const (
	diffSubtraction = "- "
	diffAddition    = "+ "
	diffUnknown     = "? "
	yellow          = "\033[93m"
	red             = "\033[91m"
	green           = "\033[92m"
	end             = "\033[0m"
)

func unifiedDiff(t *testing.T, actual, expected any) string {
	t.Helper()

	var actualString string

	var expectedString string

	switch s := actual.(type) {
	case string:
		actualString = s
	case []byte:
		actualString = string(s)
	}

	switch s := expected.(type) {
	case string:
		expectedString = s
	case []byte:
		expectedString = string(s)
	}

	d := difflibgo.Differ{}

	diffLines := d.Compare(
		strings.Split(actualString, "\n"),
		strings.Split(expectedString, "\n"),
	)

	unifiedDiffLines := make([]string, 0)

	for _, line := range diffLines {
		var diffLine string

		switch line[:2] {
		case diffUnknown:
			diffLine = yellow + line[2:] + end
		case diffSubtraction:
			diffLine = red + line[2:] + end
		case diffAddition:
			diffLine = green + line[2:] + end
		default:
			diffLine = line[2:]
		}

		unifiedDiffLines = append(unifiedDiffLines, diffLine)
	}

	return strings.Join(unifiedDiffLines, "\n")
}

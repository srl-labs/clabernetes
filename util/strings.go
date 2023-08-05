package util

import (
	"fmt"
	"strings"
)

// Indent indents all lines of a given string n spaces.
func Indent(s string, n int) string {
	ss := strings.Split(s, "\n")

	out := make([]string, len(ss))

	for idx, l := range ss {
		outLine := l

		if len(outLine) > 0 {
			outLine = fmt.Sprintf("%s%s", strings.Repeat(" ", n), l)
		}

		out[idx] = outLine
	}

	return strings.Join(out, "\n")
}

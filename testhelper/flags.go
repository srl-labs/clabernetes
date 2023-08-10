package testhelper

import (
	"flag"
)

// Update is the flag indicating if golden files should be updated when running tests.
var Update = flag.Bool("update", false, "update the golden files") //nolint: gochecknoglobals

// Flags handles parsing ortica test flags.
func Flags() {
	flag.Parse()
}

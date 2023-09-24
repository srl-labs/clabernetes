package testhelper

import (
	"flag"
)

// Update is the flag indicating if golden files should be updated when running tests.
var Update = flag.Bool("update", false, "update the golden files") //nolint: gochecknoglobals

// SkipCleanup is a bool flag that indicates if the e2e tests should skip clean up or not, by
// default this is *false* (as in we clean things up).
var SkipCleanup = flag.Bool( //nolint: gochecknoglobals
	"skipCleanup",
	false,
	"skip cleaning up namespaces",
)

// OnlyDiff is a bool flag that indicates if the test fail output should only include the diff where
// possible.
var OnlyDiff = flag.Bool( //nolint: gochecknoglobals
	"onlyDiff",
	false,
	"only show diff where possible -- skip printing out actual/expected",
)

// Flags handles parsing clabernetes test flags.
func Flags() {
	flag.Parse()
}

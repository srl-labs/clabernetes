package util

// StringSliceDifference returns the difference between a and b string slices. Items that exist in
// slice "b" but are missing in slice "a" will be returned.
func StringSliceDifference(a, b []string) []string {
	aSet := map[string]interface{}{}

	for _, s := range a {
		aSet[s] = nil
	}

	var diff []string

	for _, s := range b {
		_, found := aSet[s]
		if !found {
			diff = append(diff, s)
		}
	}

	return diff
}

// StringSliceContains returns true if the value val is in the string slice ss, otherwise false.
func StringSliceContains(ss []string, val string) bool {
	for _, s := range ss {
		if s == val {
			return true
		}
	}

	return false
}

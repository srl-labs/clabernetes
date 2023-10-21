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

// StringSliceContainsAll returns true if all values in vals are in string slice ss, otherwise
// false.
func StringSliceContainsAll(ss, vals []string) bool {
	containsAll := true

	for _, v := range vals {
		var found bool

		for _, s := range ss {
			if s == v {
				found = true

				break
			}
		}

		if !found {
			containsAll = false

			break
		}
	}

	return containsAll
}

// StringSliceEqual returns true if the string slices provided are equal, otherwise false.
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, val := range a {
		if val != b[i] {
			return false
		}
	}

	return true
}

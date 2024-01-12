package kubernetes

// ExistingMapStringStringContainsAllExpectedKeyValues returns false if the existing map of strings
// does *not* have all the keys/values from the expected/rendered labels/annotations. Basically --
// if the existing map has what we expect and anything else return true. If the existing map does
// not contain all the things we expected, return false.
func ExistingMapStringStringContainsAllExpectedKeyValues(
	existing, expected map[string]string,
) bool {
	if len(existing) == 0 && len(expected) > 0 {
		// obviously our annotations don't exist, so we need to enforce that
		return false
	}

	for k, v := range expected {
		var expectedValuesExists bool

		for nk, nv := range existing {
			if k == nk && v == nv {
				expectedValuesExists = true

				break
			}
		}

		if !expectedValuesExists {
			// missing some expected annotation, and/or value is wrong
			return false
		}
	}

	return true
}

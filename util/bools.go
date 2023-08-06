package util

// AnyBoolTrue returns True if any of the provided bool values is True.
func AnyBoolTrue(values ...bool) bool {
	for _, value := range values {
		if value {
			return true
		}
	}

	return false
}

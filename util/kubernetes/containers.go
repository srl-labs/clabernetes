package kubernetes

import (
	"reflect"

	k8scorev1 "k8s.io/api/core/v1"
)

// ContainersEqual returns true if the existing container slice matches the rendered container slice
// it ignores slice order.
func ContainersEqual(existing, rendered []k8scorev1.Container) bool {
	if len(existing) != len(rendered) {
		return false
	}

	for existingIdx := range existing {
		var matched bool

		for renderedIdx := range rendered {
			if reflect.DeepEqual(existing[existingIdx], rendered[renderedIdx]) {
				matched = true

				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

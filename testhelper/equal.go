package testhelper

import "reflect"

// MapLessDeeplyEqual is just DeepEqual but also empty/nil maps are considered equal. Mostly used
// for checking annotations in our case.
func MapLessDeeplyEqual(a, b map[string]string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	return reflect.DeepEqual(a, b)
}

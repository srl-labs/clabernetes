package util

// ToPointer accepts an object T and returns a pointer to it.
func ToPointer[T any](t T) *T {
	return &t
}

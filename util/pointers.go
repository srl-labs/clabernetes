package util

// Int32ToPointer returns i as a pointer.
func Int32ToPointer(i int32) *int32 { return &i }

// Int64ToPointer returns i as a pointer.
func Int64ToPointer(i int64) *int64 { return &i }

// BoolToPointer returns a pointer to b.
func BoolToPointer(b bool) *bool { return &b }

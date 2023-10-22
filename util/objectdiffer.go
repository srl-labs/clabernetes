package util

// ObjectDiffer holds objets of type T -- used for comparing current, missing, and extraneous
// objects in the cluster.
type ObjectDiffer[T any] struct {
	// Current objects by endpoint name
	Current map[string]T
	// Missing objects by endpoint name
	Missing []string
	// Extra objects that should be pruned
	Extra []T
}

// CurrentObjectNames returns a slice of the names of the current objects.
func (d *ObjectDiffer[T]) CurrentObjectNames() []string {
	names := make([]string, len(d.Current))

	var idx int

	for k := range d.Current {
		names[idx] = k

		idx++
	}

	return names
}

// SetMissing sets the missing objects based on the slice of all expected object names.
func (d *ObjectDiffer[T]) SetMissing(allExpectedNames []string) {
	d.Missing = StringSliceDifference(
		d.CurrentObjectNames(),
		allExpectedNames,
	)
}

// SetExtra sets the extra objects based on the slice of all expected object names and the current
// objects -- `Current` must be set prior to calling this or things will be weird.
func (d *ObjectDiffer[T]) SetExtra(allExpectedNames []string) {
	extraNames := StringSliceDifference(
		allExpectedNames,
		d.CurrentObjectNames(),
	)

	d.Extra = make([]T, len(extraNames))

	for idx, extraName := range extraNames {
		d.Extra[idx] = d.Current[extraName]
	}
}

package controllers

import (
	k8scorev1 "k8s.io/api/core/v1"
)

// ResolvedServices is an object that is used to track current and missing services for a
// controller such as Containerlab (topology).
type ResolvedServices struct {
	// Current deployments by endpoint name
	Current map[string]*k8scorev1.Service
	// Missing deployments by endpoint name
	Missing []string
	// Extra deployments that should be pruned
	Extra []*k8scorev1.Service
}

// CurrentServiceNames returns a slice of the names of the current services.
func (r *ResolvedServices) CurrentServiceNames() []string {
	names := make([]string, len(r.Current))

	var idx int

	for k := range r.Current {
		names[idx] = k

		idx++
	}

	return names
}

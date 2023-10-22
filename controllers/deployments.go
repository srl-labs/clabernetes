package controllers

import (
	"reflect"

	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
)

// ResolvedDeployments is an object that is used to track current and missing deployments for a
// controller such as Containerlab (topology).
type ResolvedDeployments struct {
	// Current deployments by endpoint name
	Current map[string]*k8sappsv1.Deployment
	// Missing deployments by endpoint name
	Missing []string
	// Extra deployments that should be pruned
	Extra []*k8sappsv1.Deployment
}

// CurrentDeploymentNames returns a slice of the names of the current deployments.
func (r *ResolvedDeployments) CurrentDeploymentNames() []string {
	names := make([]string, len(r.Current))

	var idx int

	for k := range r.Current {
		names[idx] = k

		idx++
	}

	return names
}

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

// VolumeAlreadyMounted checks if the given volumeName is already in the existingVolumes.
func VolumeAlreadyMounted(volumeName string, existingVolumes []k8scorev1.Volume) bool {
	for idx := range existingVolumes {
		if volumeName == existingVolumes[idx].Name {
			return true
		}
	}

	return false
}

package kubernetes

import k8scorev1 "k8s.io/api/core/v1"

// VolumeAlreadyMounted checks if the given volumeName is already in the existingVolumes.
func VolumeAlreadyMounted(volumeName string, existingVolumes []k8scorev1.Volume) bool {
	for idx := range existingVolumes {
		if volumeName == existingVolumes[idx].Name {
			return true
		}
	}

	return false
}

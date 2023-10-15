package config

import k8scorev1 "k8s.io/api/core/v1"

type resources struct {
	Default            *k8scorev1.ResourceRequirements                       `yaml:"default"`
	ByContainerlabKind map[string]map[string]*k8scorev1.ResourceRequirements `yaml:"byContainerlabKind"` //nolint:lll
}

func (r *resources) resourcesForContainerlabKind(
	containerlabKind, containerlabType string,
) *k8scorev1.ResourceRequirements {
	_, _ = containerlabKind, containerlabType

	return nil
}

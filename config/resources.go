package config

import (
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8scorev1 "k8s.io/api/core/v1"
)

type resources struct {
	Default            *k8scorev1.ResourceRequirements `yaml:"default"`
	ByContainerlabKind resourceMapByKindType           `yaml:"byContainerlabKind"`
}

type resourceMapByKindType map[string]map[string]*k8scorev1.ResourceRequirements

func (r *resources) resourcesForContainerlabKind(
	containerlabKind, containerlabType string,
) *k8scorev1.ResourceRequirements {
	kindResources, kindOk := r.ByContainerlabKind[containerlabKind]
	if !kindOk {
		return r.Default
	}

	explicitTypeResources, explicitTypeOk := kindResources[containerlabType]

	if explicitTypeOk {
		return explicitTypeResources
	}

	defaultTypeResources, defaultTypeOk := kindResources[clabernetesconstants.Default]

	if defaultTypeOk {
		return defaultTypeResources
	}

	return nil
}

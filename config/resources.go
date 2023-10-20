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

func (m *manager) resourcesForContainerlabKind(
	containerlabKind, containerlabType string,
) *k8scorev1.ResourceRequirements {
	m.logger.Infof(
		"looking up resources for containerlab kind %q, type %q",
		containerlabKind,
		containerlabType,
	)

	r := m.config.defaultResources

	kindResources, kindOk := r.ByContainerlabKind[containerlabKind]
	if !kindOk {
		m.logger.Debugf(
			"no kind %q found, returning default resources (if set)",
			containerlabKind,
		)

		return r.Default
	}

	explicitTypeResources, explicitTypeOk := kindResources[containerlabType]

	if explicitTypeOk {
		m.logger.Debugf(
			"explicit type %q found for kind %q, returning kind/type resources",
			containerlabType,
			containerlabKind,
		)

		return explicitTypeResources
	}

	defaultTypeResources, defaultTypeOk := kindResources[clabernetesconstants.Default]

	if defaultTypeOk {
		m.logger.Debugf(
			"no type %q found for kind %q, returning kind resources (if set)",
			containerlabType,
			containerlabKind,
		)

		return defaultTypeResources
	}

	m.logger.Debugf(
		"no default resources found for kind %q, returning default resources (if set)",
		containerlabKind,
	)

	return r.Default
}

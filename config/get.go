package config

import k8scorev1 "k8s.io/api/core/v1"

func (m *manager) GetGlobalAnnotations() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outAnnotations := make(map[string]string)

	for k, v := range m.config.globalAnnotations {
		outAnnotations[k] = v
	}

	return outAnnotations
}

func (m *manager) GetGlobalLabels() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outLabels := make(map[string]string)

	for k, v := range m.config.globalLabels {
		outLabels[k] = v
	}

	return outLabels
}

func (m *manager) GetAllMetadata() (outAnnotations, outLabels map[string]string) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	outAnnotations = make(map[string]string)

	for k, v := range m.config.globalAnnotations {
		outAnnotations[k] = v
	}

	outLabels = make(map[string]string)

	for k, v := range m.config.globalLabels {
		outLabels[k] = v
	}

	return outAnnotations, outLabels
}

func (m *manager) GetResourcesForContainerlabKind(
	containerlabKind string,
	containerlabType string,
) *k8scorev1.ResourceRequirements {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.resourcesForContainerlabKind(
		containerlabKind,
		containerlabType,
	)
}

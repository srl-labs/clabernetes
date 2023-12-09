package config

import (
	k8scorev1 "k8s.io/api/core/v1"
)

func (m *manager) GetGlobalAnnotations() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outAnnotations := make(map[string]string)

	for k, v := range m.config.Metadata.Annotations {
		outAnnotations[k] = v
	}

	return outAnnotations
}

func (m *manager) GetGlobalLabels() map[string]string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	// we dont want to pass by ref, so make a new map
	outLabels := make(map[string]string)

	for k, v := range m.config.Metadata.Labels {
		outLabels[k] = v
	}

	return outLabels
}

func (m *manager) GetAllMetadata() (outAnnotations, outLabels map[string]string) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	outAnnotations = make(map[string]string)

	for k, v := range m.config.Metadata.Annotations {
		outAnnotations[k] = v
	}

	outLabels = make(map[string]string)

	for k, v := range m.config.Metadata.Labels {
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

func (m *manager) GetPrivilegedLauncher() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.Deployment.PrivilegedLauncher
}

func (m *manager) GetContainerlabDebug() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.Deployment.ContainerlabDebug
}

func (m *manager) GetInClusterDNSSuffix() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.InClusterDNSSuffix
}

func (m *manager) GetImagePullThroughMode() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.ImagePull.PullThroughOverride
}

func (m *manager) GetImagePullCriSockOverride() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.ImagePull.CRISockOverride
}

func (m *manager) GetImagePullCriKindOverride() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.ImagePull.CRIKindOverride
}

func (m *manager) GetLauncherImage() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.Deployment.LauncherImage
}

func (m *manager) GetLauncherImagePullPolicy() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.Deployment.LauncherImagePullPolicy
}

func (m *manager) GetLauncherLogLevel() string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.config.Deployment.LauncherLogLevel
}

package config

import (
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8scorev1 "k8s.io/api/core/v1"
)

// GetFakeManager returns a fake config manager -- eventually this should have some options to load
// it with data for unit tests. That is a future me problem.
func GetFakeManager() Manager {
	return fakeManager{}
}

type fakeManager struct{}

func (f fakeManager) Start() error {
	return nil
}

func (f fakeManager) GetGlobalAnnotations() map[string]string {
	return make(map[string]string)
}

func (f fakeManager) GetGlobalLabels() map[string]string {
	return make(map[string]string)
}

func (f fakeManager) GetAllMetadata() (annotations, labels map[string]string) {
	return f.GetGlobalAnnotations(), f.GetGlobalLabels()
}

func (f fakeManager) GetResourcesForContainerlabKind(
	containerlabKind string,
	containerlabType string,
) *k8scorev1.ResourceRequirements {
	_, _ = containerlabKind, containerlabType

	return nil
}

func (f fakeManager) GetPrivilegedLauncher() bool {
	return true
}

func (f fakeManager) GetContainerlabDebug() bool {
	return false
}

func (f fakeManager) GetContainerlabTimeout() string {
	return ""
}

func (f fakeManager) GetInClusterDNSSuffix() string {
	return "svc.cluster.local"
}

func (f fakeManager) GetImagePullThroughMode() string {
	return "auto"
}

func (f fakeManager) GetLauncherImage() string {
	return "ghcr.io/srl-labs/clabernetes/clabernetes-launcher:latest"
}

func (f fakeManager) GetImagePullCriSockOverride() string {
	return ""
}

func (f fakeManager) GetImagePullCriKindOverride() string {
	return ""
}

func (f fakeManager) GetDockerDaemonConfig() string {
	return ""
}

func (f fakeManager) GetDockerConfig() string {
	return ""
}

func (f fakeManager) GetLauncherImagePullPolicy() string {
	return clabernetesconstants.KubernetesImagePullIfNotPresent
}

func (f fakeManager) GetLauncherLogLevel() string {
	return clabernetesconstants.Info
}

func (f fakeManager) GetExtraEnv() []k8scorev1.EnvVar {
	return nil
}

func (f fakeManager) GetRemoveTopologyPrefix() bool {
	return false
}

func (f fakeManager) GetContainerlabVersion() string {
	return ""
}

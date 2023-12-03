package config

import (
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

type bootstrapConfig struct {
	mergeMode                   string
	globalAnnotations           map[string]string
	globalLabels                map[string]string
	resourcesDefault            *k8scorev1.ResourceRequirements
	resourcesByContainerlabKind map[string]map[string]*k8scorev1.ResourceRequirements
	inClusterDNSSuffix          string
	imagePullThroughMode        string
}

func bootstrapFromConfigMap(inMap map[string]string) (*bootstrapConfig, error) {
	bc := &bootstrapConfig{
		mergeMode:            "merge",
		inClusterDNSSuffix:   "svc.cluster.local",
		imagePullThroughMode: "auto",
	}

	var outErrors []string

	mergeMode, mergeModeOk := inMap["mergeMode"]
	if mergeModeOk {
		bc.mergeMode = mergeMode
	}

	globalAnnotationsData, globalAnnotationsOk := inMap["globalAnnotations"]
	if globalAnnotationsOk {
		err := yaml.Unmarshal([]byte(globalAnnotationsData), &bc.globalAnnotations)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	globalLabelsData, globalLabelsOk := inMap["globalLabels"]
	if globalLabelsOk {
		err := yaml.Unmarshal([]byte(globalLabelsData), &bc.globalLabels)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	resourcesDefaultData, resourcesDefaultOk := inMap["resourcesDefault"]
	if resourcesDefaultOk {
		err := sigsyaml.Unmarshal([]byte(resourcesDefaultData), &bc.resourcesDefault)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	resourcesByKindData, resourcesByKindOk := inMap["resourcesByContainerlabKind"]
	if resourcesByKindOk {
		err := sigsyaml.Unmarshal([]byte(resourcesByKindData), &bc.resourcesDefault)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	inClusterDNSSuffix, inClusterDNSSuffixOk := inMap["inClusterDNSSuffix"]
	if inClusterDNSSuffixOk {
		bc.inClusterDNSSuffix = inClusterDNSSuffix
	}

	imagePullThroughMode, imagePullThroughModeOk := inMap["imagePullThroughMode"]
	if imagePullThroughModeOk {
		bc.imagePullThroughMode = imagePullThroughMode
	}

	var err error

	if len(outErrors) > 0 {
		errors := ""

		for idx, outError := range outErrors {
			errors += fmt.Sprintf("error %d '%s'", idx, outError)
		}

		err = fmt.Errorf("%w: %s", claberneteserrors.ErrParse, errors)
	}

	return bc, err
}

// MergeFromBootstrapConfig accepts a bootstrap config configmap and the instance of the global
// config CR and merges the bootstrap config data onto the CR. The merge operation is based on the
// config merge mode set in both the bootstrap config and the CR (with the CR setting taking
// precedence).
func MergeFromBootstrapConfig(
	bootstrapConfigMap *k8scorev1.ConfigMap,
	config *clabernetesapisv1alpha1.Config,
) error {
	bootstrap, err := bootstrapFromConfigMap(bootstrapConfigMap.Data)
	if err != nil {
		return err
	}

	if bootstrap.mergeMode == "replace" {
		mergeFromBootstrapConfigReplace(bootstrap, config)
	} else {
		// should only ever be "merge" if it isn't "replace", but either way, fallback to merge...
		mergeFromBootstrapConfigMerge(bootstrap, config)
	}

	return nil
}

func mergeFromBootstrapConfigMerge(
	bootstrap *bootstrapConfig,
	config *clabernetesapisv1alpha1.Config,
) {
	for k, v := range bootstrap.globalAnnotations {
		_, exists := config.Spec.Metadata.Annotations[k]
		if exists {
			continue
		}

		config.Spec.Metadata.Annotations[k] = v
	}

	for k, v := range bootstrap.globalLabels {
		_, exists := config.Spec.Metadata.Labels[k]
		if exists {
			continue
		}

		config.Spec.Metadata.Labels[k] = v
	}

	if config.Spec.InClusterDNSSuffix == "" {
		config.Spec.InClusterDNSSuffix = bootstrap.inClusterDNSSuffix
	}

	if config.Spec.ImagePull.PullThroughOverride == "" {
		config.Spec.ImagePull.PullThroughOverride = bootstrap.imagePullThroughMode
	}

	if config.Spec.Deployment.ResourcesDefault == nil {
		config.Spec.Deployment.ResourcesDefault = bootstrap.resourcesDefault
	}

	for k, v := range bootstrap.resourcesByContainerlabKind {
		_, exists := config.Spec.Deployment.ResourcesByContainerlabKind[k]
		if exists {
			continue
		}

		config.Spec.Deployment.ResourcesByContainerlabKind[k] = v
	}
}

func mergeFromBootstrapConfigReplace(
	bootstrap *bootstrapConfig,
	config *clabernetesapisv1alpha1.Config,
) {
	config.Spec = clabernetesapisv1alpha1.ConfigSpec{
		Metadata: clabernetesapisv1alpha1.ConfigMetadata{
			Annotations: bootstrap.globalAnnotations,
			Labels:      bootstrap.globalLabels,
		},
		InClusterDNSSuffix: bootstrap.inClusterDNSSuffix,
		ImagePull: clabernetesapisv1alpha1.ConfigImagePull{
			PullThroughOverride: bootstrap.imagePullThroughMode,
		},
		Deployment: clabernetesapisv1alpha1.ConfigDeployment{
			ResourcesDefault:            bootstrap.resourcesDefault,
			ResourcesByContainerlabKind: bootstrap.resourcesByContainerlabKind,
		},
	}
}
package config

import (
	"fmt"

	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	globalAnnotationsKey = "globalAnnotations"
	globalLabelsKey      = "globalLabels"
	defaultResourcesKey  = "defaultResources"
)

type global struct {
	globalAnnotations map[string]string
	globalLabels      map[string]string
	defaultResources  resources
}

func configFromMap(inMap map[string]string) (*global, error) {
	out := &global{
		globalAnnotations: make(map[string]string),
		globalLabels:      make(map[string]string),
		defaultResources: resources{
			Default:            nil,
			ByContainerlabKind: nil,
		},
	}

	var outErrors []string

	globalAnnotationsData, globalAnnotationsOk := inMap[globalAnnotationsKey]
	if globalAnnotationsOk {
		err := yaml.Unmarshal([]byte(globalAnnotationsData), &out.globalAnnotations)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	globalLabelsData, globalLabelsOk := inMap[globalLabelsKey]
	if globalLabelsOk {
		err := yaml.Unmarshal([]byte(globalLabelsData), &out.globalLabels)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	resourcesData, resourcesOk := inMap[defaultResourcesKey]
	if resourcesOk {
		err := sigsyaml.Unmarshal([]byte(resourcesData), &out.defaultResources)
		if err != nil {
			outErrors = append(outErrors, err.Error())
		}
	}

	var err error

	if len(outErrors) > 0 {
		errors := ""

		for idx, outError := range outErrors {
			errors += fmt.Sprintf("error %d '%s'", idx, outError)
		}

		err = fmt.Errorf("%w: %s", claberneteserrors.ErrParse, errors)
	}

	return out, err
}

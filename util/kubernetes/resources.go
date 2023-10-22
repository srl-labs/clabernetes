package kubernetes

import (
	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type resourceQuantities struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type resourceRequirements struct {
	Requests resourceQuantities `yaml:"requests"`
	Limits   resourceQuantities `yaml:"limits"`
}

func (r *resourceRequirements) toK8sResourceRequirements() *k8scorev1.ResourceRequirements {
	out := &k8scorev1.ResourceRequirements{
		Limits:   map[k8scorev1.ResourceName]resource.Quantity{},
		Requests: map[k8scorev1.ResourceName]resource.Quantity{},
	}

	if r.Requests.Memory != "" {
		out.Requests["memory"] = resource.MustParse(r.Requests.Memory)
	}

	if r.Requests.CPU != "" {
		out.Requests["cpu"] = resource.MustParse(r.Requests.CPU)
	}

	if r.Limits.Memory != "" {
		out.Limits["memory"] = resource.MustParse(r.Limits.Memory)
	}

	if r.Limits.CPU != "" {
		out.Limits["cpu"] = resource.MustParse(r.Limits.CPU)
	}

	return out
}

// YAMLToK8sResourceRequirements accepts a yaml string that looks suspiciously like k8s resources
// for a container and converts it to k8scorev1.ResourceRequirements.
func YAMLToK8sResourceRequirements(asYAML string) (*k8scorev1.ResourceRequirements, error) {
	out := &resourceRequirements{}

	err := yaml.Unmarshal([]byte(asYAML), out)
	if err != nil {
		return nil, err
	}

	return out.toK8sResourceRequirements(), nil
}

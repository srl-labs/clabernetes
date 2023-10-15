package util

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// NameMaxLen is the maximum length for a kubernetes name.
	NameMaxLen = 63
)

// CurrentNamespace returns the current kubernetes namespace as read from the KUBE_NAMESPACE env
// var, or the serviceaccount/namespace file on the instance.
func CurrentNamespace() (string, error) {
	namespaceFromEnv := os.Getenv("KUBE_NAMESPACE")
	if namespaceFromEnv != "" {
		return namespaceFromEnv, nil
	}

	namespaceFromFile, err := os.ReadFile(
		"/var/run/secrets/kubernetes.io/serviceaccount/namespace",
	)
	if err != nil {
		return "", err
	}

	return string(namespaceFromFile), nil
}

// MustCurrentNamespace returns the current kubernetes namespace or panics.
func MustCurrentNamespace() string {
	namespace, err := CurrentNamespace()
	if err != nil {
		Panic(err.Error())
	}

	return namespace
}

// SafeConcatNameKubernetes concats all provided strings into a string joined by "-" - if the final
// string is greater than 63 characters, the string will be shortened, and a hash will be used at
// the end of the string to keep it unique, but safely within allowed lengths.
func SafeConcatNameKubernetes(name ...string) string {
	return SafeConcatNameMax(name, NameMaxLen)
}

// SafeConcatNameMax concats all provided strings into a string joined by "-" - if the final string
// is greater than max characters, the string will be shortened, and a hash will be used at the end
// of the string to keep it unique, but safely within allowed lengths.
func SafeConcatNameMax(name []string, max int) string {
	finalName := strings.Join(name, "-")

	if len(finalName) <= max {
		return finalName
	}

	digest := sha256.Sum256([]byte(finalName))

	return finalName[0:max-8] + "-" + hex.EncodeToString(digest[0:])[0:7]
}

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

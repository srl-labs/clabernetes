package clabverter

import (
	"github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewConfigMapKRM(name, namespace string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}
}

func NewContainerlabKRM(name, namespace string, labels map[string]string) *v1alpha1.Containerlab {
	return &v1alpha1.Containerlab{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Containerlab",
			APIVersion: v1alpha1.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ContainerlabSpec{},
	}
}

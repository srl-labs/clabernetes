package util

import (
	"fmt"

	clabernetesmanagertypes "gitlab.com/carlmontanari/clabernetes/manager/types"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCertificatesSecret fetches the clabernetes app certificates secret.
func GetCertificatesSecret(
	c clabernetesmanagertypes.Clabernetes,
) (*k8scorev1.Secret, error) {
	client := c.GetKubeClient()

	ctx, ctxCancel := c.NewContextWithTimeout()
	defer ctxCancel()

	return client.CoreV1().Secrets(c.GetNamespace()).Get(
		ctx,
		fmt.Sprintf("%s-certificate", c.GetAppName()),
		metav1.GetOptions{},
	)
}

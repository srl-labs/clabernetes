package launcher

import (
	"time"

	clabernetesgeneratedclientset "github.com/srl-labs/clabernetes/generated/clientset"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	"k8s.io/client-go/rest"
)

const (
	clientDefaultTimeout = time.Minute
)

func mustNewKubeClabernetesClient(
	logger claberneteslogging.Instance,
) *clabernetesgeneratedclientset.Clientset {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("failed getting in cluster kubeconfig, err: %s", err)
	}

	kubeClabernetesClient, err := clabernetesgeneratedclientset.NewForConfig(kubeConfig)
	if err != nil {
		logger.Fatalf(
			"failed creating clabernetes kube client from in cluster kubeconfig, err: %s",
			err,
		)
	}

	return kubeClabernetesClient
}

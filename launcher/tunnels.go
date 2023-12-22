package launcher

import (
	"context"
	"os"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *clabernetes) getTunnels() ([]*clabernetesapisv1alpha1.PointToPointTunnel, error) {
	nodeName := os.Getenv(clabernetesconstants.LauncherNodeNameEnv)

	ctx, cancel := context.WithTimeout(c.ctx, clientDefaultTimeout)
	defer cancel()

	tunnelsCR, err := c.kubeClabernetesClient.ClabernetesV1alpha1().Connectivities(
		os.Getenv(clabernetesconstants.PodNamespaceEnv),
	).Get(
		ctx,
		os.Getenv(
			clabernetesconstants.LauncherTopologyNameEnv,
		),
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}

	nodeTunnels, ok := tunnelsCR.Spec.PointToPointTunnels[nodeName]
	if !ok {
		c.logger.Warnf(
			"no tunnels found for node %q, continuing but things may be broken",
			nodeName,
		)
	}

	return nodeTunnels, nil
}

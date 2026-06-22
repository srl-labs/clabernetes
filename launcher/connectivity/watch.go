package connectivity

import (
	"context"
	"fmt"
	"os"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesgeneratedclientset "github.com/srl-labs/clabernetes/generated/clientset"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerywatch "k8s.io/apimachinery/pkg/watch"
)

// connectivityObjectName returns the name of the Connectivity custom resource this launcher should
// read for its tunnels. In the decomposed (scalable) reconcile path the controller sets
// LauncherConnectivityNameEnv to a per-node Connectivity object; in the legacy path it is unset and
// we fall back to the topology-wide object. See docs/design/0001-scale-node-link-crds.md.
func connectivityObjectName() string {
	name := os.Getenv(clabernetesconstants.LauncherConnectivityNameEnv)
	if name != "" {
		return name
	}

	return os.Getenv(clabernetesconstants.LauncherTopologyNameEnv)
}

func watchConnectivity(
	ctx context.Context,
	logger claberneteslogging.Instance,
	clabernetesClient *clabernetesgeneratedclientset.Clientset,
	handleUpdate func(nodeTunnels []*clabernetesapisv1alpha1.PointToPointTunnel),
) {
	nodeName := os.Getenv(clabernetesconstants.LauncherNodeNameEnv)

	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", connectivityObjectName()),
		Watch:         true,
	}

	watch, err := clabernetesClient.ClabernetesV1alpha1().
		Connectivities(os.Getenv(clabernetesconstants.PodNamespaceEnv)).
		Watch(ctx, listOptions)
	if err != nil {
		logger.Fatalf("failed watching clabernetes connectivity, err: %s", err)
	}

	for event := range watch.ResultChan() {
		switch event.Type {
		case apimachinerywatch.Modified:
			logger.Info("processing connectivity modification event")

			tunnelsCR, ok := event.Object.(*clabernetesapisv1alpha1.Connectivity)
			if !ok {
				logger.Warn(
					"failed casting event object to connectivity custom resource," +
						" this is probably a bug",
				)

				continue
			}

			nodeTunnels, ok := tunnelsCR.Spec.PointToPointTunnels[nodeName]
			if !ok {
				logger.Warnf(
					"no tunnels found for node %q, continuing but things may be broken",
					nodeName,
				)
			}

			handleUpdate(nodeTunnels)
		case apimachinerywatch.Added,
			apimachinerywatch.Deleted,
			apimachinerywatch.Bookmark,
			apimachinerywatch.Error:
			logger.Warnf(
				"connectivity resource had %s event occur, ignoring...", event.Type,
			)
		}
	}
}

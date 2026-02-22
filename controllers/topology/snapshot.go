package topology

import (
	"context"
	"fmt"
	"time"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileSnapshotAnnotation checks if the Topology has the snapshotRequested annotation set to
// "true". If it does, it creates a new Snapshot CR and removes the annotation from the Topology.
func (c *Controller) ReconcileSnapshotAnnotation(
	ctx context.Context,
	topology *clabernetesapisv1alpha1.Topology,
) error {
	annotations := topology.GetAnnotations()
	if annotations == nil {
		return nil
	}

	if annotations[clabernetesconstants.AnnotationSnapshotRequested] != "true" {
		return nil
	}

	// Generate snapshot name: <topology>-<YYYYMMDD-HHMMSS>
	snapshotName := fmt.Sprintf(
		"%s-%s",
		topology.Name,
		time.Now().UTC().Format("20060102-150405"),
	)

	c.BaseController.Log.Infof(
		"snapshot requested for topology '%s/%s', creating snapshot %q",
		topology.Namespace,
		topology.Name,
		snapshotName,
	)

	snapshot := &clabernetesapisv1alpha1.Snapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:			 snapshotName,
			Namespace: topology.Namespace,
			Labels: map[string]string{
				clabernetesconstants.LabelTopologyOwner: topology.Name,
			},
		},
		Spec: clabernetesapisv1alpha1.SnapshotSpec{
			TopologyRef:			 topology.Name,
			TopologyNamespace: topology.Namespace,
		},
	}

	err := c.BaseController.Client.Create(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed creating Snapshot CR %q: %w", snapshotName, err)
	}

	// TODO: Replace the []byte with appendf
	// Remove the snapshotRequested annotation from the Topology
	patchBytes := []byte(fmt.Sprintf(
		`{"metadata":{"annotations":{%q:null}}}`,
		clabernetesconstants.AnnotationSnapshotRequested,
	))

	err = c.BaseController.Client.Patch(
		ctx,
		topology,
		ctrlruntimeclient.RawPatch(apimachinerytypes.MergePatchType, patchBytes),
	)
	if err != nil {
		c.BaseController.Log.Warnf(
			"failed removing snapshotRequested annotation from topology '%s/%s': %s",
			topology.Namespace,
			topology.Name,
			err,
		)
		// Non-fatal: snapshot was created, annotation removal is best-effort
	}

	return nil
}

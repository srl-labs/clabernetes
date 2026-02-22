package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile handles reconciliation for the Snapshot controller.
func (c *Controller) Reconcile(
	ctx context.Context,
	req ctrlruntime.Request,
) (ctrlruntime.Result, error) {
	c.BaseController.LogReconcileStart(req)

	snapshot := &clabernetesapisv1alpha1.Snapshot{}

	err := c.BaseController.Client.Get(ctx, req.NamespacedName, snapshot)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			c.BaseController.LogReconcileCompleteObjectNotExist(req)

			return ctrlruntime.Result{}, nil
		}

		c.BaseController.LogReconcileFailedGettingObject(req, err)

		return ctrlruntime.Result{}, err
	}

	if snapshot.DeletionTimestamp != nil {
		return ctrlruntime.Result{}, nil
	}

	// Skip already-terminal snapshots
	if snapshot.Status.Phase == clabernetesapisv1alpha1.SnapshotPhaseCompleted ||
		snapshot.Status.Phase == clabernetesapisv1alpha1.SnapshotPhaseFailed {
		c.BaseController.LogReconcileCompleteSuccess(req)

		return ctrlruntime.Result{}, nil
	}

	// Set phase to Running
	snapshot.Status.Phase = clabernetesapisv1alpha1.SnapshotPhaseRunning

	err = c.BaseController.Client.Status().Update(ctx, snapshot)
	if err != nil {
		c.BaseController.Log.Warnf(
			"failed updating snapshot '%s/%s' status to Running, error: %s",
			snapshot.Namespace,
			snapshot.Name,
			err,
		)

		return ctrlruntime.Result{}, err
	}

	// Look up the referenced Topology
	topologyNamespace := snapshot.Spec.TopologyNamespace
	if topologyNamespace == "" {
		topologyNamespace = snapshot.Namespace
	}

	topology := &clabernetesapisv1alpha1.Topology{}

	err = c.BaseController.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Namespace: topologyNamespace,
			Name:      snapshot.Spec.TopologyRef,
		},
		topology,
	)
	if err != nil {
		return c.failSnapshot(ctx, snapshot, fmt.Sprintf(
			"failed fetching topology '%s/%s': %s",
			topologyNamespace,
			snapshot.Spec.TopologyRef,
			err,
		))
	}

	// Collect node names from topology status
	nodeNames := make([]string, 0, len(topology.Status.NodeReadiness))
	for nodeName := range topology.Status.NodeReadiness {
		nodeNames = append(nodeNames, nodeName)
	}

	if len(nodeNames) == 0 {
		return c.failSnapshot(ctx, snapshot, "topology has no nodes in NodeReadiness status")
	}

	// For each node, exec containerlab save and collect configs
	configMapData := make(map[string]string)
	nodeConfigs := make(map[string][]string)

	for _, nodeName := range nodeNames {
		c.BaseController.Log.Infof(
			"saving node %q in topology %q",
			nodeName,
			snapshot.Spec.TopologyRef,
		)

		// Find launcher pod for this node
		podList := &k8scorev1.PodList{}

		err = c.BaseController.Client.List(
			ctx,
			podList,
			ctrlruntimeclient.InNamespace(topologyNamespace),
			ctrlruntimeclient.MatchingLabels{
				clabernetesconstants.LabelTopologyOwner: snapshot.Spec.TopologyRef,
				clabernetesconstants.LabelTopologyNode:  nodeName,
			},
		)
		if err != nil {
			c.BaseController.Log.Warnf(
				"failed listing pods for node %q: %s, skipping",
				nodeName,
				err,
			)

			continue
		}

		if len(podList.Items) == 0 {
			c.BaseController.Log.Warnf(
				"no pods found for node %q, skipping",
				nodeName,
			)

			continue
		}

		// Use the first running pod
		var targetPod *k8scorev1.Pod

		for idx := range podList.Items {
			if podList.Items[idx].Status.Phase == k8scorev1.PodRunning {
				targetPod = &podList.Items[idx]

				break
			}
		}

		if targetPod == nil {
			c.BaseController.Log.Warnf(
				"no running pod found for node %q, skipping",
				nodeName,
			)

			continue
		}

		// Run containerlab save
		saveOutput, saveErr := c.execInPod(
			ctx,
			topologyNamespace,
			targetPod.Name,
			nodeName,
			[]string{
				"sh",
				"-c",
				"cd /clabernetes && containerlab save -t topo.clab.yaml 2>&1",
			},
		)
		if saveErr != nil {
			c.BaseController.Log.Warnf(
				"containerlab save failed for node %q: %s",
				nodeName,
				saveErr,
			)
		}

		// Store save output
		saveOutputKey := fmt.Sprintf("%s/save-output", nodeName)
		configMapData[saveOutputKey] = saveOutput

		// List saved files
		savedFilesOutput, listErr := c.execInPod(
			ctx,
			topologyNamespace,
			targetPod.Name,
			nodeName,
			[]string{
				"sh",
				"-c",
				fmt.Sprintf(
					"find /clabernetes/clab-clabernetes-%s/%s/ -type f 2>/dev/null",
					nodeName,
					nodeName,
				),
			},
		)
		if listErr != nil {
			c.BaseController.Log.Warnf(
				"failed listing saved files for node %q: %s",
				nodeName,
				listErr,
			)

			continue
		}

		savedFiles := strings.Split(strings.TrimSpace(savedFilesOutput), "\n")
		var nodeFileKeys []string

		for _, filePath := range savedFiles {
			filePath = strings.TrimSpace(filePath)
			if filePath == "" {
				continue
			}

			// Read the file content
			fileContent, readErr := c.execInPod(
				ctx,
				topologyNamespace,
				targetPod.Name,
				nodeName,
				[]string{"cat", filePath},
			)
			if readErr != nil {
				c.BaseController.Log.Warnf(
					"failed reading file %q for node %q: %s",
					filePath,
					nodeName,
					readErr,
				)

				continue
			}

			// Build ConfigMap key: <nodeName>/<filename>
			fileName := filePath[strings.LastIndex(filePath, "/")+1:]
			key := fmt.Sprintf("%s/%s", nodeName, fileName)
			configMapData[key] = fileContent
			nodeFileKeys = append(nodeFileKeys, key)
		}

		nodeConfigs[nodeName] = nodeFileKeys
	}

	// Create the ConfigMap
	timestamp := time.Now().UTC().Format(time.RFC3339)

	configMap := &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshot.Name,
			Namespace: snapshot.Namespace,
			Labels: map[string]string{
				clabernetesconstants.LabelTopologyOwner: snapshot.Spec.TopologyRef,
			},
			Annotations: map[string]string{
				clabernetesconstants.AnnotationSnapshotTimestamp: timestamp,
			},
		},
		Data: configMapData,
	}

	// Set owner reference so ConfigMap is GC'd when Snapshot is deleted
	err = ctrlruntimeutil.SetOwnerReference(snapshot, configMap, c.BaseController.Client.Scheme())
	if err != nil {
		return c.failSnapshot(
			ctx,
			snapshot,
			fmt.Sprintf("failed setting owner reference on ConfigMap: %s", err),
		)
	}

	err = c.BaseController.Client.Create(ctx, configMap)
	if err != nil && !apimachineryerrors.IsAlreadyExists(err) {
		return c.failSnapshot(
			ctx,
			snapshot,
			fmt.Sprintf("failed creating ConfigMap %q: %s", snapshot.Name, err),
		)
	}

	// Update Snapshot status to Completed
	snapshot.Status.Phase = clabernetesapisv1alpha1.SnapshotPhaseCompleted
	snapshot.Status.ConfigMapRef = snapshot.Name
	snapshot.Status.Timestamp = timestamp
	snapshot.Status.NodeConfigs = nodeConfigs

	err = c.BaseController.Client.Status().Update(ctx, snapshot)
	if err != nil {
		c.BaseController.Log.Warnf(
			"failed updating snapshot '%s/%s' status to Completed, error: %s",
			snapshot.Namespace,
			snapshot.Name,
			err,
		)

		return ctrlruntime.Result{}, err
	}

	// Patch Topology annotations with snapshot info
	patchBytes := []byte(fmt.Sprintf(
		`{"metadata":{"annotations":{%q:%q,%q:%q}}}`,
		clabernetesconstants.AnnotationSnapshotTimestamp,
		timestamp,
		clabernetesconstants.AnnotationSnapshotLatest,
		snapshot.Name,
	))

	err = c.BaseController.Client.Patch(
		ctx,
		topology,
		ctrlruntimeclient.RawPatch(apimachinerytypes.MergePatchType, patchBytes),
	)
	if err != nil {
		c.BaseController.Log.Warnf(
			"failed patching topology annotations with snapshot info: %s",
			err,
		)
		// Non-fatal
	}

	c.BaseController.LogReconcileCompleteSuccess(req)

	return ctrlruntime.Result{}, nil
}

// failSnapshot sets the Snapshot status to Failed with the given message and returns.
func (c *Controller) failSnapshot(
	ctx context.Context,
	snapshot *clabernetesapisv1alpha1.Snapshot,
	message string,
) (ctrlruntime.Result, error) {
	c.BaseController.Log.Warnf("snapshot '%s/%s' failed: %s", snapshot.Namespace, snapshot.Name, message)

	snapshot.Status.Phase = clabernetesapisv1alpha1.SnapshotPhaseFailed
	snapshot.Status.Message = message

	err := c.BaseController.Client.Status().Update(ctx, snapshot)
	if err != nil {
		c.BaseController.Log.Warnf(
			"failed updating snapshot '%s/%s' status to Failed, error: %s",
			snapshot.Namespace,
			snapshot.Name,
			err,
		)
	}

	return ctrlruntime.Result{}, nil
}

// execInPod executes a command in the specified container of a pod and returns stdout output.
func (c *Controller) execInPod(
	ctx context.Context,
	namespace,
	podName,
	containerName string,
	command []string,
) (string, error) {
	req := c.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(
		&k8scorev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		},
		k8sscheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(c.BaseController.Config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed creating SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return stdout.String(), fmt.Errorf(
			"exec command failed: %w (stderr: %s)",
			err,
			stderr.String(),
		)
	}

	return stdout.String(), nil
}

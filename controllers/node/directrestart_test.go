//nolint:err113,gocognit,gocyclo // The matrix verifies persisted restart recovery and failure boundaries.
package node //nolint:testpackage // tests exercise persisted controller restart state.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	clabernetesinternaldeviceplan "github.com/clabernetes/clabernetes/internal/deviceplan"
	clabernetesinternaldirectpod "github.com/clabernetes/clabernetes/internal/directpod"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDirectRestartRecoversWhenApplicationIgnoresSignal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                                             string
		restarted, booting, fresh, legacy, connectivityFails, wantDelete bool
	}{
		{name: "ignores-signal", wantDelete: true},
		{name: "already-restarted", restarted: true},
		{name: "still-booting", booting: true},
		{name: "within-grace-period", fresh: true},
		{name: "legacy-baseline", legacy: true},
		{name: "connectivity-not-ready", legacy: true, connectivityFails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			node := planInputTestNode("router", "node-uid-a", "package-kind", "example/device:1")
			_, plan := directConnectivityTestPlan(t, node)
			deployment, err := clabernetesinternaldirectpod.Render(
				plan,
				directConnectivityRenderOptions(node),
			)
			if err != nil {
				t.Fatal(err)
			}
			deployment.Spec.Template.Annotations[clabernetesinternaldirectpod.PlanDigestAnnotation] = "sha256:" + strings.Repeat(
				"a",
				64,
			)
			deployment.Spec.Template.Annotations[clabernetesinternaldirectpod.NodeUIDAnnotation] = string(
				node.UID,
			)
			name := deployment.Spec.Template.Spec.Containers[0].Name
			pod := &k8scorev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "device-pod", Namespace: node.Namespace, UID: "pod-a",
				Labels: deployment.Spec.Template.Labels, Annotations: deployment.Spec.Template.Annotations,
			}, Spec: deployment.Spec.Template.Spec, Status: k8scorev1.PodStatus{ContainerStatuses: []k8scorev1.ContainerStatus{{
				Name: name, ContainerID: "containerd://original", State: k8scorev1.ContainerState{Running: &k8scorev1.ContainerStateRunning{}},
			}}}}
			if tc.restarted {
				pod.Status.ContainerStatuses[0].RestartCount = 1
			}
			if tc.booting {
				pod.Status.ContainerStatuses[0].State = k8scorev1.ContainerState{
					Waiting: &k8scorev1.ContainerStateWaiting{Reason: "ContainerCreating"},
				}
			}
			requestedAt := metav1.NewTime(time.Now().Add(-time.Hour))
			if tc.fresh {
				requestedAt = metav1.Now()
			}
			digest := "sha256:" + strings.Repeat("b", 64)
			// Use JSON so this also covers baselines surviving a manager restart/upgrade.
			baselineData := map[string]any{
				"planDigest": digest, "podUID": string(pod.UID),
				"requestedAt": requestedAt,
				"containers": []directRestartContainerBaseline{
					{Name: name, ContainerID: "containerd://original"},
				},
			}
			if tc.legacy {
				delete(baselineData, "requestedAt")
			}
			raw, err := json.Marshal(baselineData)
			if err != nil {
				t.Fatal(err)
			}
			configMap := &k8scorev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: "connectivity", Namespace: node.Namespace,
				Annotations: map[string]string{directRestartBaselineAnnotation: string(raw)},
			}}
			client := ctrlruntimefake.NewClientBuilder().
				WithScheme(planTestScheme(t)).
				WithObjects(pod, configMap).
				Build()
			executions := 0
			reconciler := &Reconciler{
				Client: client,
				DirectContainerExecutor: func(_ context.Context, _, _, containerName string, _ []string) error {
					if containerName == clabernetesinternaldirectpod.ConnectivityContainerName {
						if tc.connectivityFails {
							return errors.New("connectivity not ready")
						}

						return nil
					}
					executions++
					stored := &k8scorev1.ConfigMap{}
					if readErr := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(configMap), stored); readErr != nil {
						return readErr
					}
					baseline, valid := decodeDirectRestartBaseline(
						stored.Annotations[directRestartBaselineAnnotation],
					)
					if !valid || baseline.RequestedAt == nil {
						t.Fatal("restart signal sent before persisting deadline")
					}

					return nil
				},
			}
			err = reconciler.reconcileDirectLinkRestart(
				ctx,
				node,
				deployment,
				configMap,
				directConnectivityLifecycleAction{
					Mode: clabernetesinternaldeviceplan.LinkApplyRestart, PlanDigest: digest, AffectedNodeIDs: []string{string(node.UID)},
				},
				plan,
			)
			var pending *directRestartPendingError
			switch {
			case tc.connectivityFails:
				if err == nil || !strings.Contains(err.Error(), "connectivity not ready") {
					t.Fatalf("connectivity error = %v", err)
				}
			case tc.wantDelete || tc.restarted:
				if err != nil {
					t.Fatal(err)
				}
			case !errors.As(err, &pending) || pending.after <= 0:
				t.Fatalf("expected timed restart retry, got %v", err)
			}
			if (tc.booting || tc.connectivityFails || tc.wantDelete || tc.restarted) &&
				executions != 0 {
				t.Fatalf("unexpected restart executions: %d", executions)
			}
			stored := &k8scorev1.ConfigMap{}
			if err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(configMap), stored); err != nil {
				t.Fatal(err)
			}
			baseline, valid := decodeDirectRestartBaseline(
				stored.Annotations[directRestartBaselineAnnotation],
			)
			if !valid {
				t.Fatal("restart baseline lost")
			}
			if tc.connectivityFails && baseline.RequestedAt != nil {
				t.Fatal("deadline started before connectivity was ready")
			}
			if !tc.legacy && baseline.RequestedAt.Unix() != requestedAt.Unix() {
				t.Fatal("persisted restart deadline was reset")
			}
			err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(pod), &k8scorev1.Pod{})
			if tc.wantDelete && !apimachineryerrors.IsNotFound(err) {
				t.Fatalf("unresponsive application Pod was not replaced: %v", err)
			}
			if !tc.wantDelete && err != nil {
				t.Fatalf("successfully restarted Pod was deleted: %v", err)
			}
		})
	}
}

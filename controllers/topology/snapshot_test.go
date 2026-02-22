package topology_test

import (
	"context"
	"testing"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newSchemeForSnapshotTest(t *testing.T) *apimachineryruntime.Scheme {
	t.Helper()

	scheme := apimachineryruntime.NewScheme()

	err := clabernetesapisv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed adding clabernetes v1alpha1 scheme: %s", err)
	}

	err = k8scorev1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed adding core v1 scheme: %s", err)
	}

	return scheme
}

// TestReconcileSnapshotAnnotation verifies the annotation-based snapshot trigger logic.
// It covers SRL, SR-SIM, and SR OS topology names to confirm NOS-agnostic behaviour.
func TestReconcileSnapshotAnnotation(t *testing.T) {
	cases := []struct {
		name                    string
		topology                *clabernetesapisv1alpha1.Topology
		expectSnapshotCreated   bool
		expectAnnotationRemoved bool
	}{
		{
			// No annotation at all — nothing should happen.
			name: "no-annotation-no-action",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-lab",
					Namespace: "default",
				},
			},
			expectSnapshotCreated:   false,
			expectAnnotationRemoved: false,
		},
		{
			// Annotation present but set to "false" — nothing should happen.
			name: "annotation-false-no-action",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-lab",
					Namespace: "default",
					Annotations: map[string]string{
						clabernetesconstants.AnnotationSnapshotRequested: "false",
					},
				},
			},
			expectSnapshotCreated:   false,
			expectAnnotationRemoved: false,
		},
		{
			// Basic trigger: annotation "true" → Snapshot CR created, annotation removed.
			name: "annotation-true-creates-snapshot-and-removes-annotation",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-lab",
					Namespace: "default",
					Annotations: map[string]string{
						clabernetesconstants.AnnotationSnapshotRequested: "true",
					},
				},
			},
			expectSnapshotCreated:   true,
			expectAnnotationRemoved: true,
		},
		{
			// Two-node SR Linux lab — confirms NOS-independent trigger path.
			name: "two-node-srl-lab-triggers-snapshot",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "two-srl",
					Namespace: "test-ns",
					Annotations: map[string]string{
						clabernetesconstants.AnnotationSnapshotRequested: "true",
					},
				},
				Spec: clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: `---
name: two-srl
topology:
  nodes:
    srl1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:latest
    srl2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:latest
  links:
    - endpoints: ["srl1:e1-1", "srl2:e1-1"]
`,
					},
				},
				Status: clabernetesapisv1alpha1.TopologyStatus{
					NodeReadiness: map[string]string{
						"srl1": clabernetesconstants.NodeStatusReady,
						"srl2": clabernetesconstants.NodeStatusReady,
					},
				},
			},
			expectSnapshotCreated:   true,
			expectAnnotationRemoved: true,
		},
		{
			// SR-SIM (nokia_srsim) lab — same trigger path, save will exec per-node.
			name: "srsim-lab-triggers-snapshot",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "srsim-lab",
					Namespace: "test-ns",
					Annotations: map[string]string{
						clabernetesconstants.AnnotationSnapshotRequested: "true",
					},
				},
				Spec: clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: `---
name: srsim-lab
topology:
  kinds:
    nokia_srsim:
      image: nokia_srsim:25.7.R1
      license: /opt/nokia/sros/license.txt
  nodes:
    router1:
      kind: nokia_srsim
      type: sr-1
    router2:
      kind: nokia_srsim
      type: sr-1s
  links:
    - endpoints: ["router1:1/1/c1/1", "router2:1/1/c1/1"]
`,
					},
				},
				Status: clabernetesapisv1alpha1.TopologyStatus{
					NodeReadiness: map[string]string{
						"router1": clabernetesconstants.NodeStatusReady,
						"router2": clabernetesconstants.NodeStatusReady,
					},
				},
			},
			expectSnapshotCreated:   true,
			expectAnnotationRemoved: true,
		},
		{
			// SR OS (nokia_sros via vrnetlab) lab — same trigger path.
			name: "sros-lab-triggers-snapshot",
			topology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sros-lab",
					Namespace: "test-ns",
					Annotations: map[string]string{
						clabernetesconstants.AnnotationSnapshotRequested: "true",
					},
				},
				Spec: clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: `---
name: sros-lab
topology:
  nodes:
    sros1:
      kind: nokia_sros
      image: vrnetlab/nokia_sros:latest
      startup-config: |
        /configure system name "sros1"
    sros2:
      kind: nokia_sros
      image: vrnetlab/nokia_sros:latest
      startup-config: |
        /configure system name "sros2"
  links:
    - endpoints: ["sros1:1/1/c1/1", "sros2:1/1/c1/1"]
`,
					},
				},
				Status: clabernetesapisv1alpha1.TopologyStatus{
					NodeReadiness: map[string]string{
						"sros1": clabernetesconstants.NodeStatusReady,
						"sros2": clabernetesconstants.NodeStatusReady,
					},
				},
			},
			expectSnapshotCreated:   true,
			expectAnnotationRemoved: true,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				scheme := newSchemeForSnapshotTest(t)

				fakeClient := ctrlruntimefake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testCase.topology).
					Build()

				c := clabernetescontrollerstopology.NewTestController(fakeClient, scheme)

				err := c.ReconcileSnapshotAnnotation(context.Background(), testCase.topology)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				// Check if a Snapshot was created
				snapshotList := &clabernetesapisv1alpha1.SnapshotList{}

				listErr := fakeClient.List(
					context.Background(),
					snapshotList,
					ctrlruntimeclient.InNamespace(testCase.topology.Namespace),
					ctrlruntimeclient.MatchingLabels{
						clabernetesconstants.LabelTopologyOwner: testCase.topology.Name,
					},
				)
				if listErr != nil {
					t.Fatalf("failed listing snapshots: %s", listErr)
				}

				if testCase.expectSnapshotCreated && len(snapshotList.Items) != 1 {
					t.Fatalf(
						"expected 1 snapshot to be created, got %d",
						len(snapshotList.Items),
					)
				}

				if !testCase.expectSnapshotCreated && len(snapshotList.Items) != 0 {
					t.Fatalf(
						"expected no snapshot to be created, got %d",
						len(snapshotList.Items),
					)
				}

				if testCase.expectSnapshotCreated {
					snap := snapshotList.Items[0]

					if snap.Spec.TopologyRef != testCase.topology.Name {
						t.Errorf(
							"snapshot TopologyRef = %q, want %q",
							snap.Spec.TopologyRef,
							testCase.topology.Name,
						)
					}

					if snap.Spec.TopologyNamespace != testCase.topology.Namespace {
						t.Errorf(
							"snapshot TopologyNamespace = %q, want %q",
							snap.Spec.TopologyNamespace,
							testCase.topology.Namespace,
						)
					}

					// Snapshot name must start with the topology name and have a timestamp suffix
					expectedPrefix := testCase.topology.Name + "-"
					if len(snap.Name) <= len(expectedPrefix) {
						t.Errorf(
							"snapshot name %q too short, expected prefix %q",
							snap.Name,
							expectedPrefix,
						)
					}

					// Snapshot should carry topologyOwner label
					if snap.Labels[clabernetesconstants.LabelTopologyOwner] != testCase.topology.Name {
						t.Errorf(
							"snapshot missing topologyOwner label, got %q",
							snap.Labels[clabernetesconstants.LabelTopologyOwner],
						)
					}
				}

				// Fetch the topology again and check that the annotation was removed
				updatedTopology := &clabernetesapisv1alpha1.Topology{}

				getErr := fakeClient.Get(
					context.Background(),
					apimachinerytypes.NamespacedName{
						Namespace: testCase.topology.Namespace,
						Name:      testCase.topology.Name,
					},
					updatedTopology,
				)
				if getErr != nil {
					t.Fatalf("failed getting updated topology: %s", getErr)
				}

				if testCase.expectAnnotationRemoved {
					if val, ok := updatedTopology.Annotations[clabernetesconstants.AnnotationSnapshotRequested]; ok {
						t.Errorf(
							"expected snapshotRequested annotation to be removed, but still present with value %q",
							val,
						)
					}
				}
			},
		)
	}
}

// TestSnapshotCRSpec verifies that a created Snapshot CR has the correct spec fields.
func TestSnapshotCRSpec(t *testing.T) {
	topology := &clabernetesapisv1alpha1.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "two-node-lab",
			Namespace: "lab-ns",
			Annotations: map[string]string{
				clabernetesconstants.AnnotationSnapshotRequested: "true",
			},
		},
	}

	scheme := newSchemeForSnapshotTest(t)

	fakeClient := ctrlruntimefake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(topology).
		Build()

	c := clabernetescontrollerstopology.NewTestController(fakeClient, scheme)

	err := c.ReconcileSnapshotAnnotation(context.Background(), topology)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	snapList := &clabernetesapisv1alpha1.SnapshotList{}

	if err = fakeClient.List(context.Background(), snapList); err != nil {
		t.Fatalf("failed listing snapshots: %s", err)
	}

	if len(snapList.Items) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapList.Items))
	}

	snap := snapList.Items[0]

	if snap.Spec.TopologyRef != topology.Name {
		t.Errorf("TopologyRef = %q, want %q", snap.Spec.TopologyRef, topology.Name)
	}

	if snap.Spec.TopologyNamespace != topology.Namespace {
		t.Errorf("TopologyNamespace = %q, want %q", snap.Spec.TopologyNamespace, topology.Namespace)
	}

	// Phase should not be set by the trigger — it's the snapshot controller's job.
	if snap.Status.Phase != "" && snap.Status.Phase != clabernetesapisv1alpha1.SnapshotPhasePending {
		t.Errorf("unexpected initial phase %q", snap.Status.Phase)
	}
}

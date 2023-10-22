package testhelper

import (
	"testing"

	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

const (
	namespaceRandomPad = 8
)

// NormalizeKubernetesObject does some janky regex replace to remove controller generated fields
// we don't want to compare.
func NormalizeKubernetesObject(t *testing.T, object []byte) []byte {
	t.Helper()

	object = YQCommand(t, object, ".metadata.namespace = \"NAMESPACE\"")

	// delete some standard kube metadata things that will be different during tests that we dont
	// care about anyway
	object = YQCommand(t, object, "del(.metadata.creationTimestamp)")
	object = YQCommand(t, object, "del(.metadata.deletionTimestamp)")
	object = YQCommand(t, object, "del(.metadata.generation)")
	object = YQCommand(t, object, "del(.metadata.resourceVersion)")
	object = YQCommand(t, object, "del(.metadata.uid)")
	object = YQCommand(
		t,
		object,
		"del(.metadata.annotations.\"kubectl.kubernetes.io/last-applied-configuration\")",
	)

	// delete the status.conditions section and other status stuff that can be different depending
	// on the time we fetch it but doesnt actually matter to us
	object = YQCommand(t, object, "del(.status.conditions)")
	object = YQCommand(t, object, "del(.status.observedGeneration)")
	object = YQCommand(t, object, "del(.status.replicas)")
	object = YQCommand(t, object, "del(.status.readyReplicas)")
	object = YQCommand(t, object, "del(.status.availableReplicas)")
	object = YQCommand(t, object, "del(.status.unavailableReplicas)")
	object = YQCommand(t, object, "del(.status.updatedReplicas)")

	// can also see a uid on owner refs, we should only ever have one owner ref...
	object = YQCommand(t, object, "del(.metadata.ownerReferences[0].uid)")

	return object
}

// NewTestNamespace generates a namespace for a test.
func NewTestNamespace(testName string) string {
	return clabernetesutilkubernetes.SafeConcatNameKubernetes(
		"e2e",
		testName,
		clabernetesutil.RandomString(namespaceRandomPad),
	)
}

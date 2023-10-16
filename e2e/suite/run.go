package suite

import (
	"fmt"
	"testing"
	"time"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
)

const (
	updateReconcileWait    = 2 * time.Second
	eventuallyPollInterval = 3 * time.Second
	eventuallyMaxTime      = 120 * time.Second
)

// Run executes a clabernetes e2e test.
func Run(t *testing.T, steps []Step, testName string) { //nolint: thelper
	namespace := NewTestNamespace(testName)

	KubectlCreateNamespace(t, namespace)

	defer func() {
		if !*clabernetestesthelper.SkipCleanup {
			KubectlDeleteNamespace(t, namespace)
		}
	}()

	for _, step := range steps {
		t.Logf(LogStepDescr(step.Index, step.Description))

		stepFixtures := GlobStepFixtures(t, step.Index)

		for _, stepFixture := range stepFixtures {
			stepFixtureOperationType := GetStepFixtureType(t, stepFixture)

			KubectlFileOp(t, stepFixtureOperationType, namespace, stepFixture)
		}

		if *clabernetestesthelper.Update {
			// update is true, wait before fetching objects a bit to make sure things have had
			// time to reconcile fully
			time.Sleep(updateReconcileWait)
		}

		for kind, objects := range step.AssertObjects {
			for idx := range objects {
				object := step.AssertObjects[kind][idx]

				fileName := fmt.Sprintf("golden/%d-%s.%s.yaml", step.Index, kind, object.Name)

				if *clabernetestesthelper.Update {
					objectData := getter(t, namespace, kind, object.Name, object)

					clabernetestesthelper.WriteTestFixtureFile(t, fileName, objectData)

					// we just wrote the golden file of course it will match, no need to check
					break
				}

				eventually(
					t,
					eventuallyPollInterval,
					eventuallyMaxTime,
					func() []byte {
						return getter(t, namespace, kind, object.Name, object)
					},
					clabernetestesthelper.ReadTestFixtureFile(t, fileName),
				)
			}
		}

		t.Logf(LogStepSuccess(step.Index))
	}
}

func getter(t *testing.T, namespace, kind, objectName string, object AssertObject) []byte {
	t.Helper()

	objectData := KubectlGetOp(t, kind, namespace, objectName)

	if !object.SkipDefaultNormalize {
		objectData = NormalizeKubernetesObject(t, objectData)
	}

	for _, normalizeF := range object.NormalizeFuncs {
		objectData = normalizeF(t, objectData)
	}

	return objectData
}

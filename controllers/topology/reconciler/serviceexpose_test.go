package reconciler_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const renderServiceExposeTestName = "serviceexpose/render-service"

func TestRenderServiceExpose(t *testing.T) {
	cases := []struct {
		name                 string
		owningTopologyObject clabernetesapistopologyv1alpha1.TopologyCommonObject
		owningTopologyStatus *clabernetesapistopologyv1alpha1.TopologyStatus
		clabernetesConfigs   map[string]*clabernetesutilcontainerlab.Config
		nodeName             string
	}{
		{
			name: "simple",
			owningTopologyObject: &clabernetesapistopologyv1alpha1.Containerlab{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "render-service-expose-test",
					Namespace: "clabernetes",
				},
				Spec: clabernetesapistopologyv1alpha1.ContainerlabSpec{
					TopologyCommonSpec: clabernetesapistopologyv1alpha1.TopologyCommonSpec{},
					Config: `---
    name: test
    topology:
      nodes:
        srl1:
          kind: srl
          image: ghcr.io/nokia/srlinux
`,
				},
			},
			owningTopologyStatus: &clabernetesapistopologyv1alpha1.TopologyStatus{
				Tunnels:          map[string][]*clabernetesapistopologyv1alpha1.Tunnel{},
				NodeExposedPorts: map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{},
			},
			clabernetesConfigs: map[string]*clabernetesutilcontainerlab.Config{
				"srl1": {
					Name:   "srl1",
					Prefix: clabernetesutil.ToPointer(""),
					Topology: &clabernetesutilcontainerlab.Topology{
						Defaults: &clabernetesutilcontainerlab.NodeDefinition{
							Ports: []string{
								"21022:22/tcp",
								"21023:23/tcp",
								"21161:161/udp",
								"33333:57400/tcp",
								"60000:21/tcp",
								"60001:80/tcp",
								"60002:443/tcp",
								"60003:830/tcp",
								"60004:5000/tcp",
								"60005:5900/tcp",
								"60006:6030/tcp",
								"60007:9339/tcp",
								"60008:9340/tcp",
								"60009:9559/tcp",
							},
						},
						Kinds: nil,
						Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
							"srl1": {
								Kind:  "srl",
								Image: "ghcr.io/nokia/srlinux",
							},
						},
						Links: nil,
					},
					Debug: false,
				},
			},
			nodeName: "srl1",
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				reconciler := clabernetescontrollerstopologyreconciler.NewServiceExposeReconciler(
					&claberneteslogging.FakeInstance{},
					clabernetesapistopology.Containerlab,
					clabernetesconfig.GetFakeManager,
				)

				got := reconciler.Render(
					testCase.owningTopologyObject,
					testCase.owningTopologyStatus,
					testCase.clabernetesConfigs,
					testCase.nodeName,
				)

				if *clabernetestesthelper.Update {
					clabernetestesthelper.WriteTestFixtureJSON(
						t,
						fmt.Sprintf(
							"golden/%s/%s.json",
							renderServiceExposeTestName,
							testCase.name,
						),
						got,
					)

					clabernetestesthelper.WriteTestFixtureJSON(
						t,
						fmt.Sprintf(
							"golden/%s/%s-status.json",
							renderServiceExposeTestName,
							testCase.name,
						),
						testCase.owningTopologyStatus,
					)
				}

				var want k8scorev1.Service

				err := json.Unmarshal(
					clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf(
							"golden/%s/%s.json",
							renderServiceExposeTestName,
							testCase.name,
						),
					),
					&want,
				)
				if err != nil {
					t.Fatal(err)
				}

				var wantStatus *clabernetesapistopologyv1alpha1.TopologyStatus

				err = json.Unmarshal(
					clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf(
							"golden/%s/%s-status.json",
							renderServiceExposeTestName,
							testCase.name,
						),
					),
					&wantStatus,
				)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(got.Annotations, want.Annotations) {
					clabernetestesthelper.FailOutput(t, got.Annotations, want.Annotations)
				}
				if !reflect.DeepEqual(got.Labels, want.Labels) {
					clabernetestesthelper.FailOutput(t, got.Labels, want.Labels)
				}
				if !reflect.DeepEqual(got.Spec, want.Spec) {
					clabernetestesthelper.FailOutput(t, got.Spec, want.Spec)
				}

				// also check that the status got updated properly
				if !reflect.DeepEqual(testCase.owningTopologyStatus, wantStatus) {
					clabernetestesthelper.FailOutput(t, got.Spec, want.Spec)
				}
			})
	}
}

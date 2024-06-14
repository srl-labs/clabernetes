package topology_test

import (
	"encoding/json"
	"fmt"
	"testing"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const definitionProcessTestName = "definition"

func TestDefinitionProcess(t *testing.T) {
	cases := []struct {
		name                 string
		inTopology           *clabernetesapisv1alpha1.Topology
		reconcileData        *clabernetescontrollerstopology.ReconcileData
		removeTopologyPrefix bool
	}{
		{
			name: "containerlab-simple",
			inTopology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "process-containerlab-definition-test",
					Namespace: "clabernetes",
				},
				Spec: &clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: `---
    name: test
    topology:
      nodes:
        srl1:
          kind: srl
          image: ghcr.io/nokia/srlinux
        srl2:
          kind: srl
          image: ghcr.io/nokia/srlinux
      links:
        - endpoints: ["srl1:e1-1", "srl2:e1-1"]
`,
					},
				},
			},
			reconcileData: &clabernetescontrollerstopology.ReconcileData{
				Kind:           "containerlab",
				ResolvedHashes: clabernetesapisv1alpha1.ReconcileHashes{},
				ResolvedConfigs: map[string]*clabernetesutilcontainerlab.Config{
					"srl1": {},
					"srl2": {},
				},
				ResolvedTunnels: map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{
					"srl1": {},
					"srl2": {},
				},
			},
			removeTopologyPrefix: false,
		},
		{
			name: "containerlab-simple-remove-prefix",
			inTopology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "process-containerlab-definition-test",
					Namespace: "clabernetes",
				},
				Spec: &clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Containerlab: `---
    name: test
    topology:
      nodes:
        srl1:
          kind: srl
          image: ghcr.io/nokia/srlinux
        srl2:
          kind: srl
          image: ghcr.io/nokia/srlinux
      links:
        - endpoints: ["srl1:e1-1", "srl2:e1-1"]
`,
					},
				},
				Status: &clabernetesapisv1alpha1.TopologyStatus{
					RemoveTopologyPrefix: clabernetesutil.ToPointer(true),
				},
			},
			reconcileData: &clabernetescontrollerstopology.ReconcileData{
				Kind:           "containerlab",
				ResolvedHashes: clabernetesapisv1alpha1.ReconcileHashes{},
				ResolvedConfigs: map[string]*clabernetesutilcontainerlab.Config{
					"srl1": {},
					"srl2": {},
				},
				ResolvedTunnels: map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{
					"srl1": {},
					"srl2": {},
				},
			},
			removeTopologyPrefix: true,
		},
		// kne tests
		{
			name: "kne-simple",
			inTopology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "process-kne-definition-test",
					Namespace: "clabernetes",
				},
				Spec: &clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Kne: `      name: "2-srl-ixr6"
      nodes: {
        name: "srl1"
          vendor: NOKIA
          model: "ixr10"
          config:{
            file: "occonfig.cfg"
          }
          interfaces: {
            key: "e1-1"
              value: {
                name: "ethernet-1/1"
              }
          }
      }
      
      nodes: {
        name: "srl2"
          vendor: NOKIA
          model: "ixr10"
          config:{
            file: "occonfig.cfg"
          }
          interfaces: {
            key: "e1-1"
              value: {
                name: "ethernet-1/1"
              }
          }
      }
      
      links: {
        a_node: "srl1"
          a_int: "e1-1"
          z_node: "srl2"
          z_int: "e1-1"
      }
`,
					},
				},
			},
			reconcileData: &clabernetescontrollerstopology.ReconcileData{
				Kind:           "kne",
				ResolvedHashes: clabernetesapisv1alpha1.ReconcileHashes{},
				ResolvedConfigs: map[string]*clabernetesutilcontainerlab.Config{
					"srl1": {},
					"srl2": {},
				},
				ResolvedTunnels: map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{
					"srl1": {},
					"srl2": {},
				},
			},
			removeTopologyPrefix: false,
		},
		{
			name: "kne-simple-remove-prefix",
			inTopology: &clabernetesapisv1alpha1.Topology{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "process-kne-definition-test",
					Namespace: "clabernetes",
				},
				Spec: &clabernetesapisv1alpha1.TopologySpec{
					Definition: clabernetesapisv1alpha1.Definition{
						Kne: `      name: "2-srl-ixr6"
      nodes: {
        name: "srl1"
          vendor: NOKIA
          model: "ixr10"
          config:{
            file: "occonfig.cfg"
          }
          interfaces: {
            key: "e1-1"
              value: {
                name: "ethernet-1/1"
              }
          }
      }
      
      nodes: {
        name: "srl2"
          vendor: NOKIA
          model: "ixr10"
          config:{
            file: "occonfig.cfg"
          }
          interfaces: {
            key: "e1-1"
              value: {
                name: "ethernet-1/1"
              }
          }
      }
      
      links: {
        a_node: "srl1"
          a_int: "e1-1"
          z_node: "srl2"
          z_int: "e1-1"
      }
`,
					},
				},
				Status: &clabernetesapisv1alpha1.TopologyStatus{
					RemoveTopologyPrefix: clabernetesutil.ToPointer(true),
				},
			},
			reconcileData: &clabernetescontrollerstopology.ReconcileData{
				Kind:           "kne",
				ResolvedHashes: clabernetesapisv1alpha1.ReconcileHashes{},
				ResolvedConfigs: map[string]*clabernetesutilcontainerlab.Config{
					"srl1": {},
					"srl2": {},
				},
				ResolvedTunnels: map[string][]*clabernetesapisv1alpha1.PointToPointTunnel{
					"srl1": {},
					"srl2": {},
				},
			},
			removeTopologyPrefix: false,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				processor, err := clabernetescontrollerstopology.NewDefinitionProcessor(
					&claberneteslogging.FakeInstance{},
					testCase.inTopology,
					testCase.reconcileData,
					clabernetesconfig.GetFakeManager,
				)
				if err != nil {
					t.Fatal(err)
				}

				err = processor.Process()
				if err != nil {
					t.Fatal(err)
				}

				got := testCase.reconcileData

				if *clabernetestesthelper.Update {
					clabernetestesthelper.WriteTestFixtureJSON(
						t,
						fmt.Sprintf(
							"golden/%s/%s.json",
							definitionProcessTestName,
							testCase.name,
						),
						got,
					)
				}

				var want *clabernetescontrollerstopology.ReconcileData

				err = json.Unmarshal(
					clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf(
							"golden/%s/%s.json",
							definitionProcessTestName,
							testCase.name,
						),
					),
					&want,
				)
				if err != nil {
					t.Fatal(err)
				}

				clabernetestesthelper.MarshaledEqual(t, got, want)
			},
		)
	}
}

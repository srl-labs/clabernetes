package topology_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	k8scorev1 "k8s.io/api/core/v1"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
)

var defaultPorts = []string{
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
}

const renderConfigMapTestName = "configmap/render-config-map"

// TestRenderConfigMap ensures that we properly render the main tunnel/config configmap for a given
// c9s deployment (containerlab CR).
func TestRenderConfigMap(t *testing.T) {
	cases := []struct {
		name               string
		namespacedName     apimachinerytypes.NamespacedName
		clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config
		tunnels            map[string][]*clabernetesapistopologyv1alpha1.Tunnel
	}{
		{
			name: "basic-two-node-with-links",
			namespacedName: apimachinerytypes.NamespacedName{
				Name:      "test-configmap",
				Namespace: "nowhere",
			},
			clabernetesConfigs: map[string]*clabernetesutilcontainerlab.Config{
				"srl1": {
					Name:   "clabernetes-srl1",
					Prefix: clabernetesutil.ToPointer(""),
					Topology: &clabernetesutilcontainerlab.Topology{
						Defaults: &clabernetesutilcontainerlab.NodeDefinition{
							Ports: defaultPorts,
						},
						Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
							"srl1": {
								Kind: "srl",
							},
						},
						Links: []*clabernetesutilcontainerlab.LinkDefinition{
							{
								LinkConfig: clabernetesutilcontainerlab.LinkConfig{
									Endpoints: []string{
										"srl1:e1-1",
										"host:srl1-e1-1",
									},
								},
							},
						},
					},
					Debug: false,
				},
				"srl2": {
					Name:   "clabernetes-srl2",
					Prefix: clabernetesutil.ToPointer(""),
					Topology: &clabernetesutilcontainerlab.Topology{
						Defaults: &clabernetesutilcontainerlab.NodeDefinition{
							Ports: defaultPorts,
						},
						Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
							"srl2": {
								Kind: "srl",
							},
						},
						Links: []*clabernetesutilcontainerlab.LinkDefinition{
							{
								LinkConfig: clabernetesutilcontainerlab.LinkConfig{
									Endpoints: []string{
										"srl2:e1-1",
										"host:srl2-e1-1",
									},
								},
							},
						},
					},
					Debug: false,
				},
			},
			tunnels: map[string][]*clabernetesapistopologyv1alpha1.Tunnel{
				"srl1": {
					{
						ID:             1,
						LocalNodeName:  "srl1",
						RemoteName:     "unitTest-srl2-vx.clabernetes.svc.cluster.local",
						RemoteNodeName: "srl2",
						LocalLinkName:  "e1-1",
						RemoteLinkName: "e1-1",
					},
				},
				"srl2": {
					{
						ID:             1,
						LocalNodeName:  "srl2",
						RemoteName:     "unitTest-srl1-vx.clabernetes.svc.cluster.local",
						RemoteNodeName: "srl1",
						LocalLinkName:  "e1-1",
						RemoteLinkName: "e1-1",
					},
				},
			},
		},
		{
			name: "basic-two-node-no-links",
			namespacedName: apimachinerytypes.NamespacedName{
				Name:      "test-configmap",
				Namespace: "nowhere",
			},
			clabernetesConfigs: map[string]*clabernetesutilcontainerlab.Config{
				"srl1": {
					Name:   "clabernetes-srl1",
					Prefix: clabernetesutil.ToPointer(""),
					Topology: &clabernetesutilcontainerlab.Topology{
						Defaults: &clabernetesutilcontainerlab.NodeDefinition{
							Ports: defaultPorts,
						},
						Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
							"srl1": {
								Kind: "srl",
							},
						},
						Links: []*clabernetesutilcontainerlab.LinkDefinition{},
					},
					Debug: false,
				},
				"srl2": {
					Name:   "clabernetes-srl2",
					Prefix: clabernetesutil.ToPointer(""),
					Topology: &clabernetesutilcontainerlab.Topology{
						Defaults: &clabernetesutilcontainerlab.NodeDefinition{
							Ports: defaultPorts,
						},
						Nodes: map[string]*clabernetesutilcontainerlab.NodeDefinition{
							"srl2": {
								Kind: "srl",
							},
						},
						Links: []*clabernetesutilcontainerlab.LinkDefinition{},
					},
					Debug: false,
				},
			},
			tunnels: map[string][]*clabernetesapistopologyv1alpha1.Tunnel{
				"srl1": {},
				"srl2": {},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				reconciler := clabernetescontrollerstopology.NewConfigMapReconciler(
					clabernetesapistopology.Containerlab,
					clabernetesconfig.GetFakeManager,
				)

				got, err := reconciler.Render(
					testCase.namespacedName,
					testCase.clabernetesConfigs,
					testCase.tunnels,
				)
				if err != nil {
					t.Fatal(err)
				}

				if *clabernetestesthelper.Update {
					clabernetestesthelper.WriteTestFixtureJSON(
						t,
						fmt.Sprintf("golden/%s/%s.json", renderConfigMapTestName, testCase.name),
						got,
					)
				}

				var want k8scorev1.ConfigMap

				err = json.Unmarshal(
					clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf("golden/%s/%s.json", renderConfigMapTestName, testCase.name),
					),
					&want,
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
				if !reflect.DeepEqual(got.Data, want.Data) {
					clabernetestesthelper.FailOutput(t, got.Data, want.Data)
				}
			},
		)
	}
}

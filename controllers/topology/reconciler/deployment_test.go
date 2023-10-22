package reconciler_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"

	clabernetescontrollerstopologyreconciler "github.com/srl-labs/clabernetes/controllers/topology/reconciler"

	clabernetesapistopology "github.com/srl-labs/clabernetes/apis/topology"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetestesthelper "github.com/srl-labs/clabernetes/testhelper"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

const renderDeploymentTestName = "deployment/render-deployment"

func TestResolveDeployment(t *testing.T) {
	cases := []struct {
		name               string
		ownedDeployments   *k8sappsv1.DeploymentList
		clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config
		expectedCurrent    []string
		expectedMissing    []string
		expectedExtra      []*k8sappsv1.Deployment
	}{
		{
			name:               "simple",
			ownedDeployments:   &k8sappsv1.DeploymentList{},
			clabernetesConfigs: nil,
			expectedCurrent:    nil,
			expectedMissing:    nil,
			expectedExtra:      []*k8sappsv1.Deployment{},
		},
		{
			name:             "missing-nodes",
			ownedDeployments: &k8sappsv1.DeploymentList{},
			clabernetesConfigs: map[string]*clabernetesutilcontainerlab.Config{
				"node1": nil,
				"node2": nil,
			},
			expectedCurrent: nil,
			expectedMissing: []string{"node1", "node2"},
			expectedExtra:   []*k8sappsv1.Deployment{},
		},
		{
			name: "extra-nodes",
			ownedDeployments: &k8sappsv1.DeploymentList{
				Items: []k8sappsv1.Deployment{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "resolve-deployment-test",
							Namespace: "clabernetes",
							Labels: map[string]string{
								clabernetesconstants.LabelTopologyNode: "node2",
							},
						},
					},
				},
			},
			clabernetesConfigs: map[string]*clabernetesutilcontainerlab.Config{
				"node1": nil,
			},
			expectedCurrent: nil,
			expectedMissing: nil,
			expectedExtra: []*k8sappsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "resolve-deployment-test",
						Namespace: "clabernetes",
						Labels: map[string]string{
							clabernetesconstants.LabelTopologyNode: "node2",
						},
					},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				reconciler := clabernetescontrollerstopologyreconciler.NewDeploymentReconciler(
					&claberneteslogging.FakeInstance{},
					clabernetesapistopology.Containerlab,
					clabernetesconfig.GetFakeManager,
				)

				got, err := reconciler.Resolve(
					testCase.ownedDeployments,
					testCase.clabernetesConfigs,
					nil,
				)
				if err != nil {
					t.Fatal(err)
				}

				var gotCurrent []string

				for current := range got.Current {
					gotCurrent = append(gotCurrent, current)
				}

				if !clabernetesutil.StringSliceContainsAll(gotCurrent, testCase.expectedCurrent) {
					clabernetestesthelper.FailOutput(t, gotCurrent, testCase.expectedCurrent)
				}

				if !clabernetesutil.StringSliceContainsAll(got.Missing, testCase.expectedMissing) {
					clabernetestesthelper.FailOutput(t, got.Missing, testCase.expectedMissing)
				}

				if !reflect.DeepEqual(got.Extra, testCase.expectedExtra) {
					clabernetestesthelper.FailOutput(t, got.Extra, testCase.expectedExtra)
				}
			})
	}
}

func TestRenderDeployment(t *testing.T) {
	cases := []struct {
		name                 string
		owningTopologyObject clabernetesapistopologyv1alpha1.TopologyCommonObject
		clabernetesConfigs   map[string]*clabernetesutilcontainerlab.Config
		nodeName             string
	}{
		{
			name: "simple",
			owningTopologyObject: &clabernetesapistopologyv1alpha1.Containerlab{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "render-deployment-test",
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
		{
			name: "containerlab-debug",
			owningTopologyObject: &clabernetesapistopologyv1alpha1.Containerlab{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "render-deployment-test",
					Namespace: "clabernetes",
				},
				Spec: clabernetesapistopologyv1alpha1.ContainerlabSpec{
					TopologyCommonSpec: clabernetesapistopologyv1alpha1.TopologyCommonSpec{
						ContainerlabDebug: true,
					},
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
		{
			name: "launcher-log-level",
			owningTopologyObject: &clabernetesapistopologyv1alpha1.Containerlab{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "render-deployment-test",
					Namespace: "clabernetes",
				},
				Spec: clabernetesapistopologyv1alpha1.ContainerlabSpec{
					TopologyCommonSpec: clabernetesapistopologyv1alpha1.TopologyCommonSpec{
						LauncherLogLevel: "debug",
					},
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
		{
			name: "insecure-registries",
			owningTopologyObject: &clabernetesapistopologyv1alpha1.Containerlab{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "render-deployment-test",
					Namespace: "clabernetes",
				},
				Spec: clabernetesapistopologyv1alpha1.ContainerlabSpec{
					TopologyCommonSpec: clabernetesapistopologyv1alpha1.TopologyCommonSpec{
						InsecureRegistries: []string{"1.2.3.4", "potato.com"},
					},
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

				reconciler := clabernetescontrollerstopologyreconciler.NewDeploymentReconciler(
					&claberneteslogging.FakeInstance{},
					clabernetesapistopology.Containerlab,
					clabernetesconfig.GetFakeManager,
				)

				got := reconciler.Render(
					testCase.owningTopologyObject,
					testCase.clabernetesConfigs,
					testCase.nodeName,
				)

				if *clabernetestesthelper.Update {
					clabernetestesthelper.WriteTestFixtureJSON(
						t,
						fmt.Sprintf("golden/%s/%s.json", renderDeploymentTestName, testCase.name),
						got,
					)
				}

				var want k8sappsv1.Deployment

				err := json.Unmarshal(
					clabernetestesthelper.ReadTestFixtureFile(
						t,
						fmt.Sprintf("golden/%s/%s.json", renderDeploymentTestName, testCase.name),
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
				if !reflect.DeepEqual(got.Spec, want.Spec) {
					clabernetestesthelper.FailOutput(t, got.Spec, want.Spec)
				}
			})
	}
}

func TestDeploymentConforms(t *testing.T) {
	cases := []struct {
		name     string
		existing *k8sappsv1.Deployment
		rendered *k8sappsv1.Deployment
		ownerUID apimachinerytypes.UID
		conforms bool
	}{
		{
			name: "simple",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: true,
		},
		{
			name: "bad-replicas",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Replicas: clabernetesutil.ToPointer(int32(100)),
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Replicas: clabernetesutil.ToPointer(int32(1)),
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "bad-selector",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"something": "something",
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"something": "different",
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "bad-containers",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						Spec: k8scorev1.PodSpec{
							Containers: []k8scorev1.Container{
								{
									Name: "some-container",
								},
							},
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "bad-service-account",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						Spec: k8scorev1.PodSpec{
							ServiceAccountName: "something-else",
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						Spec: k8scorev1.PodSpec{
							ServiceAccountName: "default",
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "bad-restart-policy",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						Spec: k8scorev1.PodSpec{
							RestartPolicy: "Never",
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						Spec: k8scorev1.PodSpec{
							RestartPolicy: "Always",
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},

		// object meta annotations

		{
			name: "missing-clabernetes-global-annotations",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"somethingelse": "xyz",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"user-provided-global-annotation": "expected-value",
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "clabernetes-global-annotations-wrong-value",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"user-provided-global-annotation": "xyz",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"user-provided-global-annotation": "expected-value",
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "extra-annotations-ok",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"somethingelseentirely": "thisisok",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: true,
		},

		// object meta labels

		{
			name: "missing-clabernetes-global-annotations",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"somethingelse": "xyz",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-provided-global-label": "expected-value",
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "clabernetes-global-labels-wrong-value",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-provided-global-label": "xyz",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-provided-global-label": "expected-value",
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "extra-labels-ok",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"somethingelseentirely": "thisisok",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: true,
		},

		// template object meta annotations

		{
			name: "missing-clabernetes-global-annotations",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"somethingelse": "xyz",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"user-provided-global-annotation": "expected-value",
							},
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "clabernetes-global-annotations-wrong-value",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"user-provided-global-annotation": "xyz",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"user-provided-global-annotation": "expected-value",
							},
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "extra-annotations-ok",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"somethingelseentirely": "thisisok",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: true,
		},

		// template object meta labels

		{
			name: "missing-clabernetes-global-labels",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"somethingelse": "xyz",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"user-provided-global-label": "expected-value",
							},
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "clabernetes-global-labels-wrong-value",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"user-provided-global-label": "xyz",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"user-provided-global-label": "expected-value",
							},
						},
					},
				},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "extra-labels-ok",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
				Spec: k8sappsv1.DeploymentSpec{
					Template: k8scorev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"somethingelseentirely": "thisisok",
							},
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{},
			},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: true,
		},

		{
			name: "bad-owner",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("evil-imposter"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
		{
			name: "multiple-owner",
			existing: &k8sappsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID: apimachinerytypes.UID("evil-imposter"),
						},
						{
							UID: apimachinerytypes.UID("clabernetes-testing"),
						},
					},
				},
			},
			rendered: &k8sappsv1.Deployment{},
			ownerUID: apimachinerytypes.UID("clabernetes-testing"),
			conforms: false,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Logf("%s: starting", testCase.name)

				reconciler := clabernetescontrollerstopologyreconciler.NewDeploymentReconciler(
					&claberneteslogging.FakeInstance{},
					clabernetesapistopology.Containerlab,
					clabernetesconfig.GetFakeManager,
				)

				actual := reconciler.Conforms(
					testCase.existing,
					testCase.rendered,
					testCase.ownerUID,
				)
				if actual != testCase.conforms {
					clabernetestesthelper.FailOutput(t, testCase.existing, testCase.rendered)
				}
			})
	}
}

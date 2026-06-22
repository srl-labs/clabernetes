package topology_test

import (
	"strings"
	"testing"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetescontrollerstopology "github.com/srl-labs/clabernetes/controllers/topology"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expandTestTopology(name, connectivity, clab string) *clabernetesapisv1alpha1.Topology {
	return &clabernetesapisv1alpha1.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "clabernetes",
		},
		Spec: clabernetesapisv1alpha1.TopologySpec{
			Definition: clabernetesapisv1alpha1.Definition{
				Containerlab: clab,
			},
			Connectivity: connectivity,
		},
	}
}

func expandTopologyForTest(
	t *testing.T,
	topology *clabernetesapisv1alpha1.Topology,
) *clabernetescontrollerstopology.ExpandedTopology {
	t.Helper()

	expanded, err := clabernetescontrollerstopology.ExpandTopology(
		&claberneteslogging.FakeInstance{},
		topology,
		"",
		clabernetesconfig.GetFakeManager,
	)
	if err != nil {
		t.Fatalf("ExpandTopology returned unexpected error: %s", err)
	}

	return expanded
}

// TestExpandTopologyTwoNodesAndHostLink verifies that a simple two-node topology with one inter-node
// link and one host link expands to two Node objects and exactly one Link object -- host links must
// not produce a Link (they have no cross-pod tunnel; they live inside the node's sub-topology).
func TestExpandTopologyTwoNodesAndHostLink(t *testing.T) {
	topology := expandTestTopology(
		"two-node",
		clabernetesconstantsVXLAN,
		`name: two-node
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
    - endpoints: ["srl1:e3-3", "host:eth3-3"]
`,
	)

	expanded := expandTopologyForTest(t, topology)

	if len(expanded.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(expanded.Nodes))
	}

	wantNodeNames := []string{"srl1", "srl2"}
	for i, wantName := range wantNodeNames {
		node := expanded.Nodes[i]

		if node.Spec.NodeName != wantName {
			t.Fatalf("node %d: expected NodeName %q, got %q", i, wantName, node.Spec.NodeName)
		}

		if node.Spec.Kind != "containerlab" {
			t.Fatalf("node %s: expected Kind containerlab, got %q", wantName, node.Spec.Kind)
		}

		if node.Spec.Connectivity != clabernetesconstantsVXLAN {
			t.Fatalf(
				"node %s: expected Connectivity vxlan, got %q",
				wantName,
				node.Spec.Connectivity,
			)
		}

		if node.Spec.TopologyName != "two-node" {
			t.Fatalf(
				"node %s: expected TopologyName two-node, got %q",
				wantName,
				node.Spec.TopologyName,
			)
		}

		if !strings.Contains(node.Spec.Definition, wantName) {
			t.Fatalf(
				"node %s: expected Definition to contain node name, got:\n%s",
				wantName,
				node.Spec.Definition,
			)
		}

		if node.GetName() != "two-node-"+wantName {
			t.Fatalf(
				"node %s: expected object name two-node-%s, got %q",
				wantName,
				wantName,
				node.GetName(),
			)
		}
	}

	if len(expanded.Links) != 1 {
		t.Fatalf(
			"expected exactly 1 link (host link must not create a Link), got %d",
			len(expanded.Links),
		)
	}

	link := expanded.Links[0]

	if link.Spec.EndpointA != (clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl1", InterfaceName: "e1-1"}) {
		t.Fatalf("unexpected endpointA: %+v", link.Spec.EndpointA)
	}

	if link.Spec.EndpointB != (clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl2", InterfaceName: "e1-1"}) {
		t.Fatalf("unexpected endpointB: %+v", link.Spec.EndpointB)
	}

	if link.Spec.TunnelID != 1 {
		t.Fatalf("expected first allocated tunnel id to be 1, got %d", link.Spec.TunnelID)
	}

	if link.Spec.Connectivity != clabernetesconstantsVXLAN {
		t.Fatalf("expected link Connectivity vxlan, got %q", link.Spec.Connectivity)
	}

	if link.Spec.TopologyName != "two-node" {
		t.Fatalf("expected link TopologyName two-node, got %q", link.Spec.TopologyName)
	}
}

// TestExpandTopologyChainTunnelIDs verifies multi-link tunnel-id allocation and deterministic
// ordering for a three-node chain: two links should be produced with stable, distinct tunnel ids,
// and both halves of each link must collapse to a single Link object.
func TestExpandTopologyChainTunnelIDs(t *testing.T) {
	topology := expandTestTopology(
		"chain",
		clabernetesconstantsVXLAN,
		`name: chain
topology:
  nodes:
    srl1:
      kind: srl
      image: ghcr.io/nokia/srlinux
    srl2:
      kind: srl
      image: ghcr.io/nokia/srlinux
    srl3:
      kind: srl
      image: ghcr.io/nokia/srlinux
  links:
    - endpoints: ["srl1:e1-1", "srl2:e1-1"]
    - endpoints: ["srl2:e1-2", "srl3:e1-1"]
`,
	)

	expanded := expandTopologyForTest(t, topology)

	if len(expanded.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(expanded.Nodes))
	}

	if len(expanded.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(expanded.Links))
	}

	// links are sorted by tunnel id; ids are allocated deterministically starting at 1.
	wantLinks := []struct {
		a        clabernetesapisv1alpha1.LinkEndpoint
		b        clabernetesapisv1alpha1.LinkEndpoint
		tunnelID int
	}{
		{
			a:        clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl1", InterfaceName: "e1-1"},
			b:        clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl2", InterfaceName: "e1-1"},
			tunnelID: 1,
		},
		{
			a:        clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl2", InterfaceName: "e1-2"},
			b:        clabernetesapisv1alpha1.LinkEndpoint{NodeName: "srl3", InterfaceName: "e1-1"},
			tunnelID: 2,
		},
	}

	for i, want := range wantLinks {
		link := expanded.Links[i]

		if link.Spec.EndpointA != want.a {
			t.Fatalf("link %d: unexpected endpointA %+v (want %+v)", i, link.Spec.EndpointA, want.a)
		}

		if link.Spec.EndpointB != want.b {
			t.Fatalf("link %d: unexpected endpointB %+v (want %+v)", i, link.Spec.EndpointB, want.b)
		}

		if link.Spec.TunnelID != want.tunnelID {
			t.Fatalf("link %d: expected tunnel id %d, got %d", i, want.tunnelID, link.Spec.TunnelID)
		}
	}

	// allocation must be deterministic across repeated expansions.
	again := expandTopologyForTest(t, topology)
	if len(again.Links) != 2 ||
		again.Links[0].Spec.TunnelID != 1 ||
		again.Links[1].Spec.TunnelID != 2 {
		t.Fatalf("expansion is not deterministic across runs: %+v", again.Links)
	}
}

// TestExpandTopologyCarriesNodeFiles verifies that per-node bind/config files (for example an
// frr.conf and a daemons file mounted from ConfigMaps) are carried onto the correct Node object and
// not onto any other -- this is how routing configs reach a node's launcher in the decomposed model.
// The file *contents* live in their own ConfigMap objects; the Node only references them.
func TestExpandTopologyCarriesNodeFiles(t *testing.T) {
	topology := expandTestTopology(
		"with-files",
		clabernetesconstantsVXLAN,
		`name: with-files
topology:
  nodes:
    frr1:
      kind: linux
      image: quay.io/frrouting/frr
    frr2:
      kind: linux
      image: quay.io/frrouting/frr
  links:
    - endpoints: ["frr1:eth1", "frr2:eth1"]
`,
	)

	topology.Spec.Deployment.FilesFromConfigMap = map[string][]clabernetesapisv1alpha1.FileFromConfigMap{
		"frr1": {
			{
				FilePath:      "/etc/frr/frr.conf",
				ConfigMapName: "frr1-frr-conf",
				ConfigMapPath: "frr.conf",
			},
			{
				FilePath:      "/etc/frr/daemons",
				ConfigMapName: "frr1-daemons",
				ConfigMapPath: "daemons",
			},
		},
	}

	expanded := expandTopologyForTest(t, topology)

	filesByNode := map[string][]clabernetesapisv1alpha1.FileFromConfigMap{}
	for _, node := range expanded.Nodes {
		filesByNode[node.Spec.NodeName] = node.Spec.FilesFromConfigMap
	}

	if len(filesByNode["frr1"]) != 2 {
		t.Fatalf("expected frr1 to carry 2 config-map files, got %d", len(filesByNode["frr1"]))
	}

	if filesByNode["frr1"][0].FilePath != "/etc/frr/frr.conf" {
		t.Fatalf("unexpected first file path on frr1: %q", filesByNode["frr1"][0].FilePath)
	}

	if filesByNode["frr1"][0].ConfigMapName != "frr1-frr-conf" {
		t.Fatalf("unexpected configmap name on frr1: %q", filesByNode["frr1"][0].ConfigMapName)
	}

	if len(filesByNode["frr2"]) != 0 {
		t.Fatalf("expected frr2 to carry no config-map files, got %d", len(filesByNode["frr2"]))
	}
}

const clabernetesconstantsVXLAN = "vxlan"

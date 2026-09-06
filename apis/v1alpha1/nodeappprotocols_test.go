package v1alpha1_test

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

func nodeAppProtocolsSchema(t *testing.T) apiextensionsv1.JSONSchemaProps {
	t.Helper()

	raw, err := os.ReadFile(nodeCRDPath)
	if err != nil {
		t.Fatalf("failed reading node crd: %s", err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err = yaml.Unmarshal(raw, crd); err != nil {
		t.Fatalf("failed unmarshalling node crd: %s", err)
	}

	appProtocols, exists := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.
		Properties["spec"].Properties["appProtocols"]
	if !exists {
		t.Fatal("node crd spec has no appProtocols property")
	}
	if appProtocols.Items == nil || appProtocols.Items.Schema == nil {
		t.Fatal("appProtocols schema has no item schema")
	}

	return appProtocols
}

func TestNodeAppProtocolsSchema(t *testing.T) {
	schema := nodeAppProtocolsSchema(t)

	if schema.XListType == nil || *schema.XListType != "map" ||
		!reflect.DeepEqual(schema.XListMapKeys, []string{"port"}) {
		t.Fatalf(
			"appProtocols list semantics = type %v keys %v, want map keyed by port",
			schema.XListType,
			schema.XListMapKeys,
		)
	}
}

func TestNodeAppProtocolsPortPattern(t *testing.T) {
	schema := nodeAppProtocolsSchema(t)

	port := schema.Items.Schema.Properties["port"]
	portPattern := regexp.MustCompile(port.Pattern)
	for _, accepted := range []string{"1/tcp", "22/tcp", "161/udp", "57400/tcp", "65535/udp"} {
		if !portPattern.MatchString(accepted) {
			t.Errorf("port %q should be accepted", accepted)
		}
	}
	for _, rejected := range []string{
		"", "0/tcp", "65536/tcp", "022/tcp", "22", "22/TCP", "22/sctp", "10022:22/tcp",
	} {
		if portPattern.MatchString(rejected) {
			t.Errorf("port %q should be rejected", rejected)
		}
	}
}

func TestNodeAppProtocolsPattern(t *testing.T) {
	schema := nodeAppProtocolsSchema(t)

	appProtocol := schema.Items.Schema.Properties["appProtocol"]
	appProtocolPattern := regexp.MustCompile(appProtocol.Pattern)
	for _, accepted := range []string{
		"", "http", "netconf-ssh", "c9s.run/gnmi", "kubernetes.io/h2c", "example.com/Custom_1",
		strings.Repeat("a", 63),
		strings.Repeat("a", 253) + "/" + strings.Repeat("b", 63),
	} {
		if !appProtocolPattern.MatchString(accepted) {
			t.Errorf("appProtocol %q should be accepted", accepted)
		}
	}
	for _, rejected := range []string{
		"/http", "example.com/", "Example.com/http", "example_com/http", "example.com/-http",
		"example.com/http/extra", "contains whitespace",
		strings.Repeat("a", 64),
		"example.com/" + strings.Repeat("a", 64),
		strings.Repeat("a", 254) + "/http",
	} {
		if appProtocolPattern.MatchString(rejected) {
			t.Errorf("appProtocol %q should be rejected", rejected)
		}
	}

	if appProtocol.MaxLength == nil || *appProtocol.MaxLength != 317 {
		t.Fatalf(
			"appProtocol maxLength = %v, want 317",
			appProtocol.MaxLength,
		)
	}
}

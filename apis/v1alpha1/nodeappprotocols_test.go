package v1alpha1_test

import (
	"os"
	"reflect"
	"regexp"
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
	if schema.Items == nil || schema.Items.Schema == nil {
		t.Fatal("appProtocols schema has no item schema")
	}

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

	appProtocol := schema.Items.Schema.Properties["appProtocol"]
	appProtocolPattern := regexp.MustCompile(appProtocol.Pattern)
	for _, accepted := range []string{
		"", "http", "netconf-ssh", "c9s.run/gnmi", "kubernetes.io/h2c", "example.com/Custom_1",
	} {
		if !appProtocolPattern.MatchString(accepted) {
			t.Errorf("appProtocol %q should be accepted", accepted)
		}
	}
	for _, rejected := range []string{
		"/http", "example.com/", "Example.com/http", "example_com/http", "example.com/-http",
		"example.com/http/extra", "contains whitespace",
	} {
		if appProtocolPattern.MatchString(rejected) {
			t.Errorf("appProtocol %q should be rejected", rejected)
		}
	}

	if appProtocol.MaxLength == nil || *appProtocol.MaxLength != 317 ||
		len(appProtocol.XValidations) == 0 {
		t.Fatalf(
			"appProtocol bounds = maxLength %v validations %v, want qualified-name bounds",
			appProtocol.MaxLength,
			appProtocol.XValidations,
		)
	}
}

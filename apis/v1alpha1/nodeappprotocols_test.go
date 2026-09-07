package v1alpha1_test

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype"
	schemavalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

func nodeAppProtocolsCRD(t *testing.T) *apiextensionsv1.CustomResourceDefinition {
	t.Helper()

	raw, err := os.ReadFile(nodeCRDPath)
	if err != nil {
		t.Fatalf("failed reading node crd: %s", err)
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err = yaml.Unmarshal(raw, crd); err != nil {
		t.Fatalf("failed unmarshalling node crd: %s", err)
	}

	return crd
}

func nodeAppProtocolsSchema(t *testing.T) apiextensionsv1.JSONSchemaProps {
	t.Helper()

	crd := nodeAppProtocolsCRD(t)
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

func TestNodeAppProtocolsCRDValidation(t *testing.T) {
	crd := nodeAppProtocolsCRD(t)
	crd.Status.StoredVersions = []string{crd.Spec.Versions[0].Name}
	internalCRD := &apiextensions.CustomResourceDefinition{}
	if err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(
		crd, internalCRD, nil,
	); err != nil {
		t.Fatal(err)
	}
	errors := apiextensionsvalidation.ValidateCustomResourceDefinition(t.Context(), internalCRD)
	if len(errors) != 0 {
		t.Fatalf("Node CRD validation failed: %v", errors)
	}
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

func TestNodeAppProtocolsNames(t *testing.T) {
	schema := nodeAppProtocolsSchema(t)

	appProtocol := schema.Items.Schema.Properties["appProtocol"]
	internalSchema := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
		&appProtocol, internalSchema, nil,
	); err != nil {
		t.Fatal(err)
	}
	validator, _, err := schemavalidation.NewSchemaValidator(internalSchema)
	if err != nil {
		t.Fatal(err)
	}
	for _, accepted := range []string{
		"", "http", "netconf-ssh", "c9s.run/gnmi", "kubernetes.io/h2c", "example.com/Custom_1",
		"a.b/c", "my-domain.example/custom", "example.com/a_b.c-d",
		strings.Repeat("a", 63),
		strings.Repeat("a", 253) + "/" + strings.Repeat("b", 63),
	} {
		if !validator.Validate(accepted).IsValid() {
			t.Errorf("appProtocol %q should be accepted", accepted)
		}
	}
	for _, rejected := range []string{
		"/http", "example.com/", "Example.com/http", "example_com/http", "example.com/-http",
		"example.com/http/extra", "contains whitespace",
		"example..com/http", "example.-com/http", "example-.com/http",
		strings.Repeat("a", 64),
		"example.com/" + strings.Repeat("a", 64),
		strings.Repeat("a", 254) + "/http",
	} {
		if validator.Validate(rejected).IsValid() {
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

func TestNodeAppProtocolsValidation(t *testing.T) {
	schema := apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"appProtocols": nodeAppProtocolsSchema(t),
		},
	}
	internalSchema := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
		&schema, internalSchema, nil,
	); err != nil {
		t.Fatal(err)
	}
	validator, _, err := schemavalidation.NewSchemaValidator(internalSchema)
	if err != nil {
		t.Fatal(err)
	}
	structural, err := structuralschema.NewStructural(internalSchema)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		entries []any
		valid   bool
	}{
		{name: "absent", valid: true},
		{name: "empty list", entries: []any{}, valid: true},
		{name: "override", entries: []any{map[string]any{"port": "57400/tcp", "appProtocol": "kubernetes.io/h2c"}}, valid: true},
		{name: "suppression", entries: []any{map[string]any{"port": "443/tcp", "appProtocol": ""}}, valid: true},
		{name: "empty DNS label", entries: []any{map[string]any{"port": "22/tcp", "appProtocol": "example..com/http"}}},
		{name: "malformed port", entries: []any{map[string]any{"port": "22/TCP", "appProtocol": "ssh"}}},
		{name: "missing port", entries: []any{map[string]any{"appProtocol": "ssh"}}},
		{name: "missing protocol", entries: []any{map[string]any{"port": "22/tcp"}}},
		{name: "duplicate ports", entries: []any{
			map[string]any{"port": "22/tcp", "appProtocol": "ssh"},
			map[string]any{"port": "22/tcp", "appProtocol": ""},
		}},
		{name: "different transports", entries: []any{
			map[string]any{"port": "22/tcp", "appProtocol": "ssh"},
			map[string]any{"port": "22/udp", "appProtocol": ""},
		}, valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			obj := map[string]any{}
			if test.entries != nil {
				obj["appProtocols"] = test.entries
			}
			path := field.NewPath("spec")
			errors := schemavalidation.ValidateCustomResource(path, obj, validator)
			errors = append(errors, listtype.ValidateListSetsAndMaps(path, structural, obj)...)
			if (len(errors) == 0) != test.valid {
				t.Fatalf("validation errors = %v, want valid=%t", errors, test.valid)
			}
		})
	}
}

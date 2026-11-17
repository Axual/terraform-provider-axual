package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// terraform query cannot write a sensitive value, so only real secrets may be sensitive.
func TestSensitiveFields_onlyRealSecretsAreSensitive(t *testing.T) {
	tests := map[string]struct {
		resource  resource.Resource
		sensitive map[string]bool
	}{
		"principal":  {&applicationPrincipalResource{}, map[string]bool{"principal": false, "private_key": true}},
		"deployment": {&applicationDeploymentResource{}, map[string]bool{"configs": true, "definition": false, "sql_script": false}},
		"cluster": {&clusterResource{}, map[string]bool{
			"broker.certificate": false, "broker.private_key": true, "broker.username": false, "broker.password": true,
			"broker.truststore": false, "strimzi.provisioner_username": false, "strimzi.provisioner_password": true,
		}},
		"instance": {&instanceResource{}, map[string]bool{"ca_certificates": false}},
		"instance cluster": {&instanceClusterResource{}, map[string]bool{
			"schema_registry.username": false, "schema_registry.password": true, "schema_registry.certificate": false,
			"schema_registry.private_key": true, "schema_registry.truststore": false,
			"schema_registry.keycloak.admin_username": false, "schema_registry.keycloak.admin_password": true,
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := resource.SchemaResponse{}
			test.resource.Schema(context.Background(), resource.SchemaRequest{}, &resp)
			for attribute, want := range test.sensitive {
				if got := attributeAt(t, resp.Schema.Attributes, attribute).IsSensitive(); got != want {
					t.Errorf("%s sensitive = %v, expected %v", attribute, got, want)
				}
			}
		})
	}
}

// attributeAt finds an attribute by its path, such as "broker.private_key", through single nested attributes.
func attributeAt(t *testing.T, attributes map[string]schema.Attribute, path string) schema.Attribute {
	t.Helper()
	name, rest, nested := strings.Cut(path, ".")
	attribute, ok := attributes[name]
	if !ok {
		t.Fatalf("no attribute %q", name)
	}
	if !nested {
		return attribute
	}
	parent, ok := attribute.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("%q is not a single nested attribute", name)
	}
	return attributeAt(t, parent.Attributes, rest)
}

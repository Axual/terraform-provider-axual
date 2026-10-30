package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// terraform query cannot write a sensitive value, so only real secrets may be sensitive.
func TestSensitiveFields_onlyRealSecretsAreSensitive(t *testing.T) {
	tests := map[string]struct {
		resource  resource.Resource
		sensitive map[string]bool
	}{
		"principal":  {&applicationPrincipalResource{}, map[string]bool{"principal": false, "private_key": true}},
		"deployment": {&applicationDeploymentResource{}, map[string]bool{"configs": true, "definition": false, "sql_script": false}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := resource.SchemaResponse{}
			test.resource.Schema(context.Background(), resource.SchemaRequest{}, &resp)
			for attribute, want := range test.sensitive {
				if got := resp.Schema.Attributes[attribute].IsSensitive(); got != want {
					t.Errorf("%s sensitive = %v, expected %v", attribute, got, want)
				}
			}
		})
	}
}

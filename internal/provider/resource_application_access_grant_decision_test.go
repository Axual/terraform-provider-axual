package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDecisionValidateConfig: the configuration sets grant_id (single repository) or all four lookup
// attributes (multi-repo), never a mix.
func TestDecisionValidateConfig(t *testing.T) {
	lookups := map[string]any{"application": "Log Scraper", "topic": "t", "environment": "e", "access_type": "CONSUMER"}
	tests := []struct {
		name    string
		set     map[string]any
		wantErr bool
	}{
		{"four lookup attributes", lookups, false},
		{"grant_id", map[string]any{"grant_id": "g"}, false},
		{"grant_id known only at apply", map[string]any{"grant_id": tftypes.UnknownValue}, false},
		{"lookup attribute known only at apply", map[string]any{"application": tftypes.UnknownValue, "topic": "t", "environment": "e", "access_type": "CONSUMER"}, false},
		{"three lookup attributes", map[string]any{"application": "Log Scraper", "topic": "t", "environment": "e"}, true},
		{"neither", map[string]any{}, true},
		{"grant_id and lookup attributes", map[string]any{"grant_id": "g", "application": "Log Scraper", "topic": "t", "environment": "e", "access_type": "CONSUMER"}, true},
		{"grant_id and one lookup attribute", map[string]any{"grant_id": "g", "topic": "t"}, true},
	}
	r := &grantDecisionResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]tftypes.Value{}
			for name, attrType := range objectType.AttributeTypes {
				values[name] = tftypes.NewValue(attrType, nil)
			}
			values["decision"] = tftypes.NewValue(tftypes.String, decisionApproved)
			for name, v := range tt.set {
				values[name] = tftypes.NewValue(tftypes.String, v)
			}
			req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)}}
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Errorf("error = %v, want %v: %v", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

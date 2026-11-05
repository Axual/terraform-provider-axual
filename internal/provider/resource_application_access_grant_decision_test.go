package provider

import (
	"context"
	"testing"

	webclient "axual-webclient"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDecisionValidateConfig: the configuration sets grant_id (single repository), all four lookup
// attributes (multi-repo), or both (as `terraform plan -generate-config-out` writes them), never a part.
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
		{"grant_id and lookup attributes", map[string]any{"grant_id": "g", "application": "Log Scraper", "topic": "t", "environment": "e", "access_type": "CONSUMER"}, false},
		{"grant_id and three lookup attributes", map[string]any{"grant_id": "g", "application": "Log Scraper", "topic": "t", "environment": "e"}, true},
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

// TestGrantMatchesLookups: with grant_id and the lookup attributes, both must describe the same grant.
func TestGrantMatchesLookups(t *testing.T) {
	g := webclient.GrantView{Uid: "g1", TopicUid: "t", EnvironmentUid: "e", AccessType: "CONSUMER",
		ApplicationName: "Log Scraper", ApplicationUid: "0123456789abcdef0123456789abcdef"}
	d := func(app, topic, env, access string) grantDecisionData {
		return grantDecisionData{Application: types.StringValue(app), Topic: types.StringValue(topic),
			Environment: types.StringValue(env), AccessType: types.StringValue(access)}
	}
	tests := []struct {
		name string
		d    grantDecisionData
		want bool
	}{
		{"by name", d("Log Scraper", "t", "e", "CONSUMER"), true},
		{"by UID, other case", d("0123456789ABCDEF0123456789ABCDEF", "t", "e", "CONSUMER"), true},
		{"access type in lower case", d("Log Scraper", "t", "e", "consumer"), true},
		{"other application", d("POC Secret", "t", "e", "CONSUMER"), false},
		{"name in other case", d("log scraper", "t", "e", "CONSUMER"), false},
		{"other topic", d("Log Scraper", "t2", "e", "CONSUMER"), false},
		{"other environment", d("Log Scraper", "t", "e2", "CONSUMER"), false},
		{"other access type", d("Log Scraper", "t", "e", "PRODUCER"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grantMatchesLookups(g, tt.d); got != tt.want {
				t.Errorf("grantMatchesLookups = %v, want %v", got, tt.want)
			}
		})
	}
}

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// schemaVersionConfig is a configuration of the axual_schema_version data source with the given owners.
func schemaVersionConfig(t *testing.T, owners tftypes.Value) tfsdk.Config {
	t.Helper()
	var schemaResp datasource.SchemaResponse
	(&schemaVersionDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	values["full_name"] = tftypes.NewValue(tftypes.String, "io.axual.Test")
	values["version"] = tftypes.NewValue(tftypes.String, "1.0.0")
	values["owners"] = owners
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)}
}

// The data source does not filter by owner, so a configured owners only gives a warning. Reading owners
// gives none.
func TestSchemaVersionDataSourceWarnsWhenOwnersIsSet(t *testing.T) {
	for name, tc := range map[string]struct {
		owners      tftypes.Value
		wantWarning bool
	}{
		"owners set":      {tftypes.NewValue(tftypes.String, "group-uid"), true},
		"owners unknown":  {tftypes.NewValue(tftypes.String, tftypes.UnknownValue), true},
		"owners left out": {tftypes.NewValue(tftypes.String, nil), false},
	} {
		t.Run(name, func(t *testing.T) {
			resp := datasource.ValidateConfigResponse{}
			req := datasource.ValidateConfigRequest{Config: schemaVersionConfig(t, tc.owners)}
			(&schemaVersionDataSource{}).ValidateConfig(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() || (resp.Diagnostics.WarningsCount() == 1) != tc.wantWarning {
				t.Fatalf("diagnostics = %v, expected a warning: %v", resp.Diagnostics, tc.wantWarning)
			}
		})
	}
}

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTopicConfigRetentionTime(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewTopicConfigResource(AxualProvider{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	attr := resp.Schema.Attributes["retention_time"].(schema.Int64Attribute)
	for value, want := range map[int64]bool{-2: false, -1: true, 0: false, 999: false, 1000: true, 160704000000: true, 160704000001: false} {
		var diags diag.Diagnostics
		for _, v := range attr.Validators {
			r := validator.Int64Response{}
			v.ValidateInt64(ctx, validator.Int64Request{Path: path.Root("retention_time"), ConfigValue: types.Int64Value(value)}, &r)
			diags.Append(r.Diagnostics...)
		}
		if got := !diags.HasError(); got != want {
			t.Errorf("retention_time %d valid = %v, expected %v", value, got, want)
		}
		if n := diags.ErrorsCount(); n > 1 {
			t.Errorf("retention_time %d gives %d errors, expected one", value, n)
		}
	}
}

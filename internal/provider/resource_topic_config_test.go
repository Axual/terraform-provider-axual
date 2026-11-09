package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

func TestSupportsInfiniteRetention(t *testing.T) {
	for version, want := range map[string]bool{
		"15.0.8": false, "16.0.2": false, "16.1.0": true, "16.1.0-SNAPSHOT": true, "16.10.0": true, "17.0.0": true, "": false, "unknown": false,
	} {
		if got := SupportsInfiniteRetention(version); got != want {
			t.Errorf("SupportsInfiniteRetention(%q) = %v, expected %v", version, got, want)
		}
	}
}

// The plan reads GET /info only when it sets retention_time = -1 on a topic config that does not have it.
func TestTopicConfigPlanRefusesInfiniteRetentionOnOlderPlatformManager(t *testing.T) {
	for _, tc := range []struct {
		name      string
		info      string // GET /info answer; "" answers 500
		state     int64  // 0: the plan creates the topic config
		retention int64
		wantError bool
		wantCalls int
	}{
		{"create on 16.0.2", `{"version":"16.0.2"}`, 0, -1, true, 1},
		{"update on 15.0.8", `{"version":"15.0.8"}`, 604800000, -1, true, 1},
		{"create on 16.1.0", `{"version":"16.1.0-SNAPSHOT"}`, 0, -1, false, 1},
		{"version unreadable", "", 0, -1, true, 1},
		{"already -1", `{"version":"16.0.2"}`, -1, -1, false, 0},
		{"not -1", `{"version":"16.0.2"}`, 0, 604800000, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/info" || tc.info == "" {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, _ = w.Write([]byte(tc.info))
			})
			r := NewTopicConfigResource(AxualProvider{client: client})
			planned := stateFor(t, r, topicConfigWithRetention(tc.retention))
			req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}, State: emptyState(t, r)}
			if tc.state != 0 {
				req.State = stateFor(t, r, topicConfigWithRetention(tc.state))
			}
			resp := resource.ModifyPlanResponse{Plan: req.Plan}
			r.(resource.ResourceWithModifyPlan).ModifyPlan(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError || calls != tc.wantCalls {
				t.Errorf("error = %v, GET /info calls = %d; expected error %v, %d calls. Diagnostics: %v",
					resp.Diagnostics.HasError(), calls, tc.wantError, tc.wantCalls, resp.Diagnostics)
			}
		})
	}
}

func topicConfigWithRetention(retention int64) *topicConfigResourceData {
	return &topicConfigResourceData{RetentionTime: types.Int64Value(retention), Properties: types.MapNull(types.StringType)}
}

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSameJSON(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{`{"type":"record","name":"A"}`, "{\n  \"name\": \"A\",\n  \"type\": \"record\"\n}", true},
		{`{"name":"A"}`, `{"name":"B"}`, false},
		{`syntax = "proto3";`, `syntax = "proto3";`, false},                     // not JSON
		{`{"default":9007199254740993}`, `{"default":9007199254740992}`, false}, // a big number keeps its digits
		{`{"default":1.50}`, `{"default":1.50}`, true},
	}
	for _, tt := range tests {
		if got := sameJSON(tt.a, tt.b); got != tt.want {
			t.Errorf("sameJSON(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// A body with the same JSON in other whitespace plans the stored body, so there is no change.
func TestSameJSONBodyKeepsTheStoredBody(t *testing.T) {
	stored := `{"type":"record","name":"A","fields":[]}`
	tests := []struct {
		name, config, want string
	}{
		{"other whitespace", "{\n  \"type\": \"record\",\n  \"name\": \"A\",\n  \"fields\": []\n}", stored},
		{"other JSON", `{"type":"record","name":"B","fields":[]}`, `{"type":"record","name":"B","fields":[]}`},
		{"not JSON", `syntax = "proto3";`, `syntax = "proto3";`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: types.StringValue(stored), ConfigValue: types.StringValue(test.config), PlanValue: types.StringValue(test.config)}
			resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
			sameJSONBody{}.PlanModifyString(context.Background(), req, &resp)
			if resp.PlanValue.ValueString() != test.want {
				t.Errorf("planned %q, expected %q", resp.PlanValue.ValueString(), test.want)
			}
		})
	}
}

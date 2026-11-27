package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

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

// A deleted schema version (404, or PM's 400 "... converted to null") is removed from the state. Other errors fail the read.
func TestSchemaVersionReadDropsAGoneVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		status      int
		body        string
		wantRemoved bool
	}{
		"400 converted to null": {http.StatusBadRequest, `{"detail":"Required URI template variable 'uid' for method parameter type is present but converted to null","status":400}`, true},
		"404":                   {http.StatusNotFound, "", true},
		"another 400":           {http.StatusBadRequest, `{"detail":"Invalid request"}`, false},
		"500":                   {http.StatusInternalServerError, `{"detail":"boom"}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/schema_versions/sv1" {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			r := NewSchemaVersionResource(AxualProvider{client: client})
			state := stateFor(t, r, &schemaVersionResourceData{Id: types.StringValue("sv1")})
			resp := resource.ReadResponse{State: state}

			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

			if tc.wantRemoved && (resp.Diagnostics.HasError() || !resp.State.Raw.IsNull()) {
				t.Errorf("read = %v, state null = %t; expected the schema version removed from the state", resp.Diagnostics, resp.State.Raw.IsNull())
			}
			if !tc.wantRemoved && (!resp.Diagnostics.HasError() || resp.State.Raw.IsNull()) {
				t.Errorf("read = %v, state null = %t; expected an error and the state kept", resp.Diagnostics, resp.State.Raw.IsNull())
			}
		})
	}
}

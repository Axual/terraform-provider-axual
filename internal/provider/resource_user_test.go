package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	webclient "axual-webclient"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// No roles must reach PATCH /users/{id}/roles as [], not null; the endpoint refuses
// null with "Required request body is missing".
func TestUserRequestSendsAnEmptyRolesList(t *testing.T) {
	body, err := json.Marshal(createUserRequestFromData(context.Background(), &userResourceData{}, nil).Roles)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("roles body = %s, expected []", body)
	}
}

// A user with no roles is stored as an empty set, not null, so `roles = []` stays consistent.
func TestMapUserKeepsAnEmptyRoleSet(t *testing.T) {
	data := userResourceData{}
	mapUserResponseToData(context.Background(), &data, &webclient.UserResponse{})
	if data.Roles.IsNull() || len(data.Roles.Elements()) != 0 {
		t.Errorf("roles = %v, expected an empty set", data.Roles)
	}
}

// A 3.2.x state of a user with no roles has roles = null. Applied with -refresh=false, the plan keeps
// null: the user keeps its roles, and the state keeps the planned null, or Terraform rejects the result.
func TestUserUpdateKeepsTheRolesWhenThePlanHasNone(t *testing.T) {
	var sentRoles string
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/roles"):
			body, _ := io.ReadAll(r.Body)
			sentRoles = string(body)
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"uid":"u1","firstName":"Jane","lastName":"Doe","emailAddress":{"email":"jane@example.com"},"roles":[{"name":"STREAM_AUTHOR"}]}`))
		default:
			_, _ = w.Write([]byte(`{"uid":"u1","firstName":"Janet","lastName":"Doe","emailAddress":{"email":"jane@example.com"},"roles":[{"name":"STREAM_AUTHOR"}]}`))
		}
	})
	r := &userResource{provider: AxualProvider{client: client}}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	rolesType := objectType.AttributeTypes["roles"]
	raw := func(firstName string, roles tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objectType, map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "u1"),
			"first_name":    tftypes.NewValue(tftypes.String, firstName),
			"middle_name":   tftypes.NewValue(tftypes.String, nil),
			"last_name":     tftypes.NewValue(tftypes.String, "Doe"),
			"email_address": tftypes.NewValue(tftypes.String, "jane@example.com"),
			"phone_number":  tftypes.NewValue(tftypes.String, nil),
			"roles":         roles,
		})
	}
	tests := map[string]struct {
		planned       tftypes.Value
		expectedState string
	}{
		"null":    {tftypes.NewValue(rolesType, nil), "<null>"},
		"unknown": {tftypes.NewValue(rolesType, tftypes.UnknownValue), `[{"name":"STREAM_AUTHOR"}]`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sentRoles = ""
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: raw("Jane", tftypes.NewValue(rolesType, nil))}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw("Janet", test.planned)}
			resp := resource.UpdateResponse{State: state}

			r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)

			var saved userResourceData
			resp.State.Get(ctx, &saved)
			if resp.Diagnostics.HasError() || sentRoles != `[{"name":"STREAM_AUTHOR"}]` || saved.Roles.String() != test.expectedState {
				t.Errorf("update = %v, sent roles %s, saved roles %s; expected the user's current roles sent and %s saved",
					resp.Diagnostics, sentRoles, saved.Roles, test.expectedState)
			}
		})
	}
}

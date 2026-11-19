package provider

import (
	webclient "axual-webclient"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDeriveTargetFromTypes(t *testing.T) {
	tests := map[string]struct {
		types  []string
		target string
	}{
		"schema registry basic auth": {[]string{"SCHEMA_REGISTRY_BASIC_AUTH"}, "SCHEMA_REGISTRY"},
		"schema registry oauth":      {[]string{"SCHEMA_REGISTRY_OAUTH"}, "SCHEMA_REGISTRY"},
		"scram":                      {[]string{"SCRAM_SHA_512"}, "KAFKA"},
		"no types":                   {nil, "KAFKA"},
		"schema registry and scram":  {[]string{"SCRAM_SHA_512", "SCHEMA_REGISTRY_OAUTH"}, "SCHEMA_REGISTRY"},
	}
	for name, test := range tests {
		var authTypes []webclient.AuthType
		for _, authType := range test.types {
			authTypes = append(authTypes, webclient.AuthType{Type: authType})
		}
		if got := deriveTargetFromTypes(authTypes); got != test.target {
			t.Errorf("%s: deriveTargetFromTypes = %s, expected %s", name, got, test.target)
		}
	}
}

// A Schema Registry credential generated again outside Terraform keeps its username and gets a new ID.
// A refresh follows it through the framework's own identity check, and warns once.
func TestApplicationCredentialRefreshFollowsAReplacedCredential(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		storedID, password string
		wantWarning        string // a part of the warning; empty for no warning
	}{
		"same credential":             {"cred-new", "secret", ""},
		"replaced, made by Terraform": {"cred-old", "secret", "does not work any more"},
		"replaced, imported":          {"cred-old", "", "follows the new credential"},
		"state without an ID":         {"", "secret", ""},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`[{"id":"cred-new","username":"appenv","application":{"id":"app"},"environment":{"id":"env"},
					"types":[{"type":"SCHEMA_REGISTRY_BASIC_AUTH"}],"description":"d","metadata":{"clusters":"c"}}]`))
			})
			server := providerserver.NewProtocol6(&AxualProvider{client: client})()
			r := withIdentity(NewApplicationCredentialResource(AxualProvider{client: client}), "id")
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			var identityResp resource.IdentitySchemaResponse
			r.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
			stateType := schemaResp.Schema.Type().TerraformType(ctx)
			identityType := identityResp.IdentitySchema.Type().TerraformType(ctx)

			state := stateFor(t, r, applicationCredentialResourceData{
				Id: types.StringValue(tc.storedID), ApplicationId: types.StringValue("app"), EnvironmentId: types.StringValue("env"),
				Target: types.StringValue("SCHEMA_REGISTRY"), UserName: types.StringValue("appenv"), Password: types.StringValue(tc.password),
				Clusters: types.StringValue("c"), Description: types.StringValue("d"), AuthProvider: types.StringValue("apicurio"),
			})
			currentState, err := tfprotov6.NewDynamicValue(stateType, state.Raw)
			if err != nil {
				t.Fatal(err)
			}
			req := &tfprotov6.ReadResourceRequest{TypeName: "axual_application_credential", CurrentState: &currentState}
			if tc.storedID != "" {
				identity, err := tfprotov6.NewDynamicValue(identityType, tftypes.NewValue(identityType, map[string]tftypes.Value{
					"id": tftypes.NewValue(tftypes.String, tc.storedID)}))
				if err != nil {
					t.Fatal(err)
				}
				req.CurrentIdentity = &tfprotov6.ResourceIdentityData{IdentityData: &identity}
			}

			resp, err := server.ReadResource(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			var warnings []string
			for _, d := range resp.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("error: %s: %s", d.Summary, d.Detail)
				}
				warnings = append(warnings, d.Detail)
			}
			if tc.wantWarning == "" && len(warnings) > 0 || tc.wantWarning != "" && (len(warnings) != 1 || !strings.Contains(warnings[0], tc.wantWarning)) {
				t.Errorf("warnings = %q, expected one with %q", warnings, tc.wantWarning)
			}
			if tc.password == "" && len(warnings) == 1 && strings.Contains(warnings[0], "password") {
				t.Errorf("a credential without a password in the state must not get the password advice: %q", warnings[0])
			}
			newState, err := resp.NewState.Unmarshal(stateType)
			if err != nil {
				t.Fatal(err)
			}
			newIdentity, err := resp.NewIdentity.IdentityData.Unmarshal(identityType)
			if err != nil {
				t.Fatal(err)
			}
			for what, value := range map[string]tftypes.Value{"state": newState, "identity": newIdentity} {
				var attributes map[string]tftypes.Value
				var id string
				if err := value.As(&attributes); err != nil || attributes["id"].As(&id) != nil || id != "cred-new" {
					t.Errorf("%s id = %q, expected cred-new", what, id)
				}
			}
		})
	}
}

// After the refresh that found a replaced credential, the state has the new ID, so the plans after it
// find no change. The warning about the dead password still repeats on each of them, through the
// private state, until the resource is replaced. A credential without a password never gets it.
func TestApplicationCredentialStalePasswordWarnsOnEveryPlan(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		password      string
		wantLaterWarn bool
	}{
		"made by Terraform": {"secret", true},
		"imported":          {"", false},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`[{"id":"cred-new","username":"appenv","application":{"id":"app"},"environment":{"id":"env"},
					"types":[{"type":"SCHEMA_REGISTRY_BASIC_AUTH"}],"description":"d","metadata":{"clusters":"c"}}]`))
			})
			server := providerserver.NewProtocol6(&AxualProvider{client: client})()
			r := withIdentity(NewApplicationCredentialResource(AxualProvider{client: client}), "id")
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			var identityResp resource.IdentitySchemaResponse
			r.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
			stateType, identityType := schemaResp.Schema.Type().TerraformType(ctx), identityResp.IdentitySchema.Type().TerraformType(ctx)
			state := stateFor(t, r, applicationCredentialResourceData{
				Id: types.StringValue("cred-old"), ApplicationId: types.StringValue("app"), EnvironmentId: types.StringValue("env"),
				Target: types.StringValue("SCHEMA_REGISTRY"), UserName: types.StringValue("appenv"), Password: types.StringValue(tc.password),
				Clusters: types.StringValue("c"), Description: types.StringValue("d"), AuthProvider: types.StringValue("apicurio"),
			})
			current, _ := tfprotov6.NewDynamicValue(stateType, state.Raw)
			identity, _ := tfprotov6.NewDynamicValue(identityType, tftypes.NewValue(identityType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "cred-old")}))
			req := &tfprotov6.ReadResourceRequest{TypeName: "axual_application_credential", CurrentState: &current,
				CurrentIdentity: &tfprotov6.ResourceIdentityData{IdentityData: &identity}}
			warnings := func(resp *tfprotov6.ReadResourceResponse) (n int) {
				for _, d := range resp.Diagnostics {
					if d.Severity == tfprotov6.DiagnosticSeverityError {
						t.Fatalf("error: %s: %s", d.Summary, d.Detail)
					}
					n++
				}
				return n
			}
			first, err := server.ReadResource(ctx, req)
			if err != nil || warnings(first) != 1 {
				t.Fatalf("first refresh: %v, %v; expected the replaced warning", err, first.Diagnostics)
			}
			for i := 2; i <= 3; i++ {
				next, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: "axual_application_credential",
					CurrentState: first.NewState, CurrentIdentity: first.NewIdentity, Private: first.Private})
				if err != nil {
					t.Fatal(err)
				}
				if got := warnings(next) == 1; got != tc.wantLaterWarn {
					t.Errorf("refresh %d: warnings %v, expected a warning %v", i, next.Diagnostics, tc.wantLaterWarn)
				}
				first = next
			}
		})
	}
}

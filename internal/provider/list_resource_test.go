package provider

import (
	webclient "axual-webclient"
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// readableResource reads every ID except "bad", for which Read fails.
type readableResource struct{ removedResource }

func (readableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	req.State.GetAttribute(ctx, path.Root("id"), &id)
	if id.ValueString() == "bad" {
		resp.Diagnostics.AddError("Client Error", "status: 500")
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), id)
}

func (readableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func TestListSkipsUnreadableResource(t *testing.T) {
	warnings, ids := runReadableList(t, 1, []listFound{{id: "bad", displayName: "a"}, {id: "good", displayName: "b"}})
	// "Found 2 axual_readable, returning the first 1" and "Skipped axual_readable bad"; the skipped one
	// does not use up the limit of 1.
	if len(warnings) != 2 || len(ids) != 1 || ids[0] != "good" {
		t.Errorf("warnings = %v, results = %v; want 2 warnings and good", warnings, ids)
	}
}

// Resources the user may not manage are left out with one warning that names them. They do not use
// up the limit.
func TestListWarnsAboutForbiddenResources(t *testing.T) {
	warnings, ids := runReadableList(t, 1, []listFound{
		{id: "f1", displayName: "orders on dev", forbidden: true},
		{id: "good", displayName: "payments on dev"},
		{id: "f2", displayName: "users on prd", forbidden: true},
	})
	if len(ids) != 1 || ids[0] != "good" {
		t.Errorf("results = %v, want only good", ids)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Skipped 2 axual_readable") ||
		!strings.Contains(warnings[0], "orders on dev, users on prd") {
		t.Errorf("warnings = %v, want one that names both forbidden resources", warnings)
	}
}

// runReadableList lists found with readableResource and returns the warnings (summary and detail)
// and the IDs of the results.
func runReadableList(t *testing.T, limit int64, found []listFound) ([]string, []string) {
	t.Helper()
	ctx := context.Background()
	l := &axualListResource{
		provider: AxualProvider{client: &webclient.Client{}},
		spec: listSpec{
			typeName:    "axual_readable",
			attr:        "id",
			newResource: func(AxualProvider) resource.Resource { return readableResource{} },
			find: func(context.Context, *webclient.Client, map[string]string) ([]listFound, error) {
				return found, nil
			},
		},
	}
	w := withIdentity(readableResource{}, "id").(*identityResource)
	var schemaResp resource.SchemaResponse
	w.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	var identityResp resource.IdentitySchemaResponse
	w.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
	var configResp list.ListResourceSchemaResponse
	l.ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, &configResp)

	req := list.ListRequest{
		Config:                 tfsdk.Config{Schema: configResp.Schema, Raw: tftypes.NewValue(configResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{})},
		IncludeResource:        true,
		Limit:                  limit,
		ResourceSchema:         schemaResp.Schema,
		ResourceIdentitySchema: identityResp.IdentitySchema,
	}
	var stream list.ListResultsStream
	l.List(ctx, req, &stream)

	var warnings, ids []string
	for r := range stream.Results {
		if r.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", r.Diagnostics)
		}
		if r.Identity == nil {
			for _, d := range r.Diagnostics.Warnings() {
				warnings = append(warnings, d.Summary()+": "+d.Detail())
			}
			continue
		}
		var id types.String
		r.Identity.GetAttribute(ctx, path.Root("id"), &id)
		ids = append(ids, id.ValueString())
	}
	return warnings, ids
}

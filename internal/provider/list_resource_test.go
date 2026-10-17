package provider

import (
	webclient "axual-webclient"
	"context"
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
	ctx := context.Background()
	l := &axualListResource{
		provider: AxualProvider{client: &webclient.Client{}},
		spec: listSpec{
			typeName:    "axual_readable",
			attr:        "id",
			newResource: func(AxualProvider) resource.Resource { return readableResource{} },
			find: func(context.Context, *webclient.Client, map[string]string) ([]listFound, error) {
				return []listFound{{id: "bad", displayName: "a"}, {id: "good", displayName: "b"}}, nil
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
		Limit:                  1,
		ResourceSchema:         schemaResp.Schema,
		ResourceIdentitySchema: identityResp.IdentitySchema,
	}
	var stream list.ListResultsStream
	l.List(ctx, req, &stream)

	var warnings, results int
	for r := range stream.Results {
		if r.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", r.Diagnostics)
		}
		if r.Identity == nil {
			warnings += len(r.Diagnostics.Warnings())
			continue
		}
		results++
		var id types.String
		r.Identity.GetAttribute(ctx, path.Root("id"), &id)
		if id.ValueString() != "good" {
			t.Errorf("result id = %q, want good", id.ValueString())
		}
	}
	// "Only 1 of 2" and "Skipped ... bad"; the skipped one does not use up the limit of 1.
	if warnings != 2 || results != 1 {
		t.Errorf("warnings = %d, results = %d; want 2 and 1", warnings, results)
	}
}

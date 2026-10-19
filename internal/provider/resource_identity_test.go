package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// removedResource is a resource whose Read always finds it deleted.
type removedResource struct{}

func (removedResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "axual_removed"
}
func (removedResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}}}
}
func (removedResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (removedResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.State.RemoveResource(ctx)
}
func (removedResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (removedResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}
func (removedResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
}

func TestIdentityAfterReadOfRemovedResource(t *testing.T) {
	ctx := context.Background()
	w := withIdentity(removedResource{}, "id").(*identityResource)

	var schemaResp resource.SchemaResponse
	w.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	var identityResp resource.IdentitySchemaResponse
	w.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.SetAttribute(ctx, path.Root("id"), "abc"); diags.HasError() {
		t.Fatalf("set id: %v", diags)
	}
	// No prior identity, as in an older state.
	identity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema, Raw: tftypes.NewValue(identityResp.IdentitySchema.Type().TerraformType(ctx), nil)}

	resp := resource.ReadResponse{State: state, Identity: identity}
	w.Read(ctx, resource.ReadRequest{State: state, Identity: identity}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("the resource should be removed from the state")
	}
	var id types.String
	resp.Identity.GetAttribute(ctx, path.Root("id"), &id)
	if id.ValueString() != "abc" {
		t.Errorf("identity id = %q, want abc", id.ValueString())
	}
}

// The wrapper only passes on ImportState and ModifyPlan. A wrapped resource that implements any other
// optional interface would lose it without an error.
func TestIdentityWrapperHidesNothing(t *testing.T) {
	for _, f := range (&AxualProvider{}).Resources(context.Background()) {
		w, ok := f().(*identityResource)
		if !ok {
			continue
		}
		if _, ok := w.Resource.(resource.ResourceWithImportState); !ok {
			t.Errorf("%T: needs ImportState, the wrapper calls it", w.Resource)
		}
		hidden := map[string]bool{}
		_, hidden["ValidateConfig"] = w.Resource.(resource.ResourceWithValidateConfig)
		_, hidden["ConfigValidators"] = w.Resource.(resource.ResourceWithConfigValidators)
		_, hidden["UpgradeState"] = w.Resource.(resource.ResourceWithUpgradeState)
		_, hidden["MoveState"] = w.Resource.(resource.ResourceWithMoveState)
		_, hidden["Configure"] = w.Resource.(resource.ResourceWithConfigure)
		for name, isHidden := range hidden {
			if isHidden {
				t.Errorf("%T: %s is hidden by identityResource; forward it in the wrapper", w.Resource, name)
			}
		}
	}
}

// The identity attribute of each list resource is a string attribute of its resource.
func TestListSpecsIdentityIsAStringAttribute(t *testing.T) {
	ctx := context.Background()
	for _, s := range listSpecs() {
		var resp resource.SchemaResponse
		s.newResource(AxualProvider{}).Schema(ctx, resource.SchemaRequest{}, &resp)
		if _, ok := resp.Schema.Attributes[s.attr].(schema.StringAttribute); !ok {
			t.Errorf("%s: identity attribute %q is not a string attribute of the resource", s.typeName, s.attr)
		}
	}
}

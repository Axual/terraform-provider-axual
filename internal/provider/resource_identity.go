package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithImportState = &identityResource{}
var _ resource.ResourceWithIdentity = &identityResource{}
var _ resource.ResourceWithModifyPlan = &identityResource{}

// identityResource adds a one-attribute identity (the import ID) to a resource, which `terraform
// query` needs. It copies the identity from state after Create, Read, Update and ImportState.
type identityResource struct {
	resource.Resource
	attr string
}

// withIdentity wraps r so that it has a resource identity named attr.
// The wrapped resource must implement resource.ResourceWithImportState.
func withIdentity(r resource.Resource, attr string) resource.Resource {
	return &identityResource{Resource: r, attr: attr}
}

func (w *identityResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			w.attr: identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "The ID used to import this resource.",
			},
		},
	}
}

func (w *identityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	w.Resource.Create(ctx, req, resp)
	w.copyIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (w *identityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	w.Resource.Read(ctx, req, resp)
	if resp.State.Raw.IsNull() {
		// Removed, but the framework still needs an identity, also for a state without one (3.2.0).
		w.copyIdentity(ctx, req.State, resp.Identity, &resp.Diagnostics)
		return
	}
	w.copyIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (w *identityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	w.Resource.Update(ctx, req, resp)
	w.copyIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (w *identityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" && req.Identity != nil {
		var id types.String
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root(w.attr), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.ID = id.ValueString()
	}
	w.Resource.(resource.ResourceWithImportState).ImportState(ctx, req, resp)
	w.copyIdentity(ctx, resp.State, resp.Identity, &resp.Diagnostics)
}

func (w *identityResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r, ok := w.Resource.(resource.ResourceWithModifyPlan); ok {
		r.ModifyPlan(ctx, req, resp)
	}
}

func (w *identityResource) copyIdentity(ctx context.Context, state tfsdk.State, identity *tfsdk.ResourceIdentity, diags *diag.Diagnostics) {
	if identity == nil || state.Raw.IsNull() || diags.HasError() {
		return
	}
	var id types.String
	diags.Append(state.GetAttribute(ctx, path.Root(w.attr), &id)...)
	if id.IsNull() || id.IsUnknown() {
		return
	}
	diags.Append(identity.SetAttribute(ctx, path.Root(w.attr), id)...)
}

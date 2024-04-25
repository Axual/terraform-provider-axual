package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationAccessGrantRejectionResource{}
var _ resource.ResourceWithImportState = &applicationAccessGrantRejectionResource{}

func NewApplicationAccessGrantRejectionResource(provider AxualProvider) resource.Resource {
	return &applicationAccessGrantRejectionResource{
		provider: provider,
	}
}

type applicationAccessGrantRejectionResource struct {
	provider AxualProvider
}

type GrantRejectionData struct {
	ApplicationAccessGrant types.String `tfsdk:"application_access_grant"`
	Reason                 types.String `tfsdk:"reason"`
}

func (r *applicationAccessGrantRejectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant_rejection"
}

func (r *applicationAccessGrantRejectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Application Access Grant Rejection: Reject a request to access a topic`,
		Attributes: map[string]schema.Attribute{
			"application_access_grant": schema.StringAttribute{
				MarkdownDescription: "Application Access Grant Unique Identifier.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "Reason for denying approval.",
				Optional:            true,
			},
		},
	}
}

func (r *applicationAccessGrantRejectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GrantRejectionData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Links.Deny.Href != "" {

		err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.ValueString(), data.Reason.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to reject grant", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		tflog.Info(ctx, "Saving Application Access Grant Rejection resource to state")
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}

	// If Grant is already Rejected, simply import the state
	if applicationAccessGrant.Status == "Rejected" {
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		tflog.Info(ctx, "mapping the resource")
	}

	resp.Diagnostics.AddError(
		"Error: Failed to Reject/Deny grant",
		fmt.Sprintf("Grant is not in correct state \nCurrent status of the grant is: %s", applicationAccessGrant.Status))

}

func (r *applicationAccessGrantRejectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GrantRejectionData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	// If Grant is already Rejected, simply import the state
	if applicationAccessGrant.Status == "Rejected" {
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		tflog.Info(ctx, "mapping the resource")
		return
	}
	resp.Diagnostics.AddError("Grant is not Rejected", "Error")

}

func (r *applicationAccessGrantRejectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant Rejection cannot be Edited",
		"Please delete the Application Access Grant Approval to revoke Approval",
	)
}

func (r *applicationAccessGrantRejectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// We should be able to delete a rejection in whatever state it's in.
}

func (r *applicationAccessGrantRejectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

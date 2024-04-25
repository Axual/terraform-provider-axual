package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationAccessGrantApprovalResource{}
var _ resource.ResourceWithImportState = &applicationAccessGrantApprovalResource{}

func NewApplicationAccessGrantApprovalResource(provider AxualProvider) resource.Resource {
	return &applicationAccessGrantApprovalResource{
		provider: provider,
	}
}

type applicationAccessGrantApprovalResource struct {
	provider AxualProvider
}
type GrantApprovalData struct {
	ApplicationAccessGrant types.String `tfsdk:"application_access_grant"`
}

func (r *applicationAccessGrantApprovalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant_approval"
}

func (r *applicationAccessGrantApprovalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Application Access Grant Approval: Approve access to a topic`,
		Attributes: map[string]schema.Attribute{
			"application_access_grant": schema.StringAttribute{
				MarkdownDescription: "Application Access Grant Unique Identifier.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *applicationAccessGrantApprovalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GrantApprovalData

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

	if applicationAccessGrant.Links.Approve.Href != "" {
		tflog.Info(ctx, "Approving Application Access Grant")
		err := r.provider.client.ApproveGrant(data.ApplicationAccessGrant.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to approve grant", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		tflog.Info(ctx, "Saving Application Access Grant Approval resource to state")
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}

	// If Grant is already approved, simply import the state
	if applicationAccessGrant.Status == "Approved" {
		tflog.Info(ctx, "Saving Application Access Grant Approval resource to state")
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.AddError(
		"Error: Failed to approve grant",
		fmt.Sprintf("Only Pending grants can be approved \nCurrent status of the grant is: %s", applicationAccessGrant.Status))
}

func (r *applicationAccessGrantApprovalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GrantApprovalData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application Access Grant not found. Id: %s", data.ApplicationAccessGrant.ValueString()))
		} else {
			resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	if applicationAccessGrant.Status == "Approved" {
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		tflog.Info(ctx, "mapping the resource")
		data.ApplicationAccessGrant = types.StringValue(applicationAccessGrant.Uid)
	} else {
		resp.Diagnostics.AddError("Grant is not Approved", fmt.Sprintf("Only Pending grants can be approved \nCurrent status of the grant is: %s", applicationAccessGrant.Status))
		return
	}
}

func (r *applicationAccessGrantApprovalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant Approval cannot be Edited",
		"Please delete the Approval to revoke Approval, then create a new Approval for a different grant",
	)
}

func (r *applicationAccessGrantApprovalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var data GrantApprovalData

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

	if applicationAccessGrant.Links.Revoke.Href != "" {
		err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.ValueString(), "Revoked in terraform")
		if err != nil {
			resp.Diagnostics.AddError("Failed to revoke approval for application access grant", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.AddError(
		"Error: Failed to Revoke grant",
		fmt.Sprintf("Only Approved grants can be revoked \n Current status of the grant is: %s", applicationAccessGrant.Status))

}

func (r *applicationAccessGrantApprovalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("application_access_grant"), req, resp)
}

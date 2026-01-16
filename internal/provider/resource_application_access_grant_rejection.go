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
		tflog.Info(ctx, "Grant is already Rejected, adopting into Terraform state")
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Specific error messages for non-rejectable states with remediation steps
	grantId := data.ApplicationAccessGrant.ValueString()
	switch applicationAccessGrant.Status {
	case "Approved":
		resp.Diagnostics.AddError(
			"Cannot reject approved grant",
			fmt.Sprintf(
				"Grant '%s' is already approved. Approved grants cannot be rejected.\n\n"+
					"To deny access:\n"+
					"1. Delete the axual_application_access_grant_approval resource to revoke the grant\n"+
					"2. Or delete the axual_application_access_grant resource (auto-revokes)\n\n"+
					"Tip: Run 'terraform state show axual_application_access_grant.<name>' to check the grant's current status.",
				grantId))
	case "Revoked":
		resp.Diagnostics.AddError(
			"Cannot reject revoked grant",
			fmt.Sprintf(
				"Grant '%s' was previously approved and then revoked. "+
					"Revoked grants cannot be rejected.\n\n"+
					"The grant is already in a terminal state - access is denied.\n"+
					"To request access again, the Application Owner must delete and recreate the grant.\n\n"+
					"Tip: Run 'terraform state show axual_application_access_grant.<name>' to check the grant's current status.",
				grantId))
	case "Cancelled":
		resp.Diagnostics.AddError(
			"Cannot reject cancelled grant",
			fmt.Sprintf(
				"Grant '%s' is cancelled by the Application Owner. "+
					"Cancelled grants cannot be rejected.\n\n"+
					"The grant is already in a terminal state - the access request was withdrawn.\n\n"+
					"Tip: Run 'terraform state show axual_application_access_grant.<name>' to check the grant's current status.",
				grantId))
	default:
		resp.Diagnostics.AddError(
			"Cannot reject grant",
			fmt.Sprintf(
				"Only Pending grants can be rejected.\n"+
					"Current status: %s\nGrant ID: %s\n\n"+
					"Tip: Run 'terraform state show axual_application_access_grant.<name>' to check the grant's current status.",
				applicationAccessGrant.Status, grantId))
	}

}

func (r *applicationAccessGrantRejectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GrantRejectionData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Reading Application Access Grant Rejection for grant: %s", data.ApplicationAccessGrant.ValueString()))

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application Access Grant not found, removing rejection from state. Id: %s", data.ApplicationAccessGrant.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Status == "Rejected" {
		tflog.Info(ctx, fmt.Sprintf("Grant is Rejected, saving rejection state. Id: %s", data.ApplicationAccessGrant.ValueString()))
		// Map the API's 'comment' field to the Terraform 'reason' attribute
		if applicationAccessGrant.Comment != "" {
			data.Reason = types.StringValue(applicationAccessGrant.Comment)
		}
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Grant exists but is not in Rejected status
	// Remove the rejection resource from state since it no longer represents a rejected grant
	tflog.Warn(ctx, fmt.Sprintf("Grant is not in Rejected status (current: %s), removing rejection from state. Id: %s",
		applicationAccessGrant.Status, data.ApplicationAccessGrant.ValueString()))
	resp.State.RemoveResource(ctx)
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
	resource.ImportStatePassthroughID(ctx, path.Root("application_access_grant"), req, resp)
}

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = applicationAccessGrantApprovalResourceType{}
var _ tfsdk.Resource = applicationAccessGrantApprovalResource{}
var _ tfsdk.ResourceWithImportState = applicationAccessGrantApprovalResource{}

type applicationAccessGrantApprovalResourceType struct{}

func (t applicationAccessGrantApprovalResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {

	return tfsdk.Schema{
		MarkdownDescription: `Application Access Grant Approval: Approve access to a stream`,
		Attributes: map[string]tfsdk.Attribute{
			"application_access_grant": {
				MarkdownDescription: "Application Access Grant Unique Identifier.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (t applicationAccessGrantApprovalResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationAccessGrantApprovalResource{
		provider: provider,
	}, diags
}

type GrantApprovalData struct {
	ApplicationAccessGrant types.String `tfsdk:"application_access_grant"`
}

type applicationAccessGrantApprovalResource struct {
	provider provider
}

func (r applicationAccessGrantApprovalResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data GrantApprovalData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Links.Approve.Href != "" {
		tflog.Info(ctx, "Approving Application Access Grant")
		err := r.provider.client.ApproveGrant(data.ApplicationAccessGrant.Value)
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

func (r applicationAccessGrantApprovalResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data GrantApprovalData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Status == "Approved" {
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		tflog.Info(ctx, "mapping the resource")
		data.ApplicationAccessGrant = types.String{Value: applicationAccessGrant.Uid}
	} else {
		resp.Diagnostics.AddError("Grant is not Approved", fmt.Sprintf("Only Pending grants can be approved \nCurrent status of the grant is: %s", applicationAccessGrant.Status))
		return
	}
}

func (r applicationAccessGrantApprovalResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant Approval cannot be Edited",
		"Please delete the Approval to revoke Approval, then create a new Approval for a different grant",
	)
}

func (r applicationAccessGrantApprovalResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {

	var data GrantApprovalData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Links.Revoke.Href != "" {
		err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, "Revoked in terraform")
		if err != nil {
			resp.Diagnostics.AddError("Failed to revoke approval for application access grant", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
	}

	resp.Diagnostics.AddError(
		"Error: Failed to Revoke grant",
		fmt.Sprintf("Only Approved grants can be revoked \n Current status of the grant is: %s", applicationAccessGrant.Status))

}

func (r applicationAccessGrantApprovalResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("application_access_grant"), req, resp)
}

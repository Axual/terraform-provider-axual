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

var _ tfsdk.ResourceType = applicationAccessGrantRejectionResourceType{}
var _ tfsdk.Resource = applicationAccessGrantRejectionResource{}
var _ tfsdk.ResourceWithImportState = applicationAccessGrantRejectionResource{}

type applicationAccessGrantRejectionResourceType struct{}

func (t applicationAccessGrantRejectionResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {

	return tfsdk.Schema{
		MarkdownDescription: `Application Access Grant Rejection: Reject a request to access a topic`,
		Attributes: map[string]tfsdk.Attribute{
			"application_access_grant": {
				MarkdownDescription: "Application Access Grant Unique Identifier.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"reason": {
				MarkdownDescription: "Reason for denying approval.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (t applicationAccessGrantRejectionResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationAccessGrantRejectionResource{
		provider: provider,
	}, diags
}

type GrantRejectionData struct {
	ApplicationAccessGrant types.String `tfsdk:"application_access_grant"`
	Reason                 types.String `tfsdk:"reason"`
}

type applicationAccessGrantRejectionResource struct {
	provider provider
}

func (r applicationAccessGrantRejectionResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data GrantRejectionData

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

	if applicationAccessGrant.Links.Deny.Href != "" {

		err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, data.Reason.Value)
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

func (r applicationAccessGrantRejectionResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data GrantRejectionData

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
	// If Grant is already Rejected, simply import the state
	if applicationAccessGrant.Status == "Rejected" {
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		tflog.Info(ctx, "mapping the resource")
		return
	}
	resp.Diagnostics.AddError("Grant is not Rejected", "Error")

}

func (r applicationAccessGrantRejectionResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant Rejection cannot be Edited",
		"Please delete the Application Access Grant Approval to revoke Approval",
	)
}

func (r applicationAccessGrantRejectionResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// We should be able to delete a rejection in whatever state it's in.
}

func (r applicationAccessGrantRejectionResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("application_access_grant"), req, resp)
}

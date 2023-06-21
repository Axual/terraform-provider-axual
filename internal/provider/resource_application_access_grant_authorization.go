package provider

import (
	"context"
	"fmt"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = applicationAccessGrantAuthorizationResourceType{}
var _ tfsdk.Resource = applicationAccessGrantAuthorizationResource{}
var _ tfsdk.ResourceWithImportState = applicationAccessGrantAuthorizationResource{}

type applicationAccessGrantAuthorizationResourceType struct{}

func (t applicationAccessGrantAuthorizationResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {

	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Application Access Grant Authorization. Set the status of an Access Grant",
		Attributes: map[string]tfsdk.Attribute{
			"status": {
				MarkdownDescription: "Status of Application Access Grant.",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Approved", "Revoked", "Rejected", "Pending"}),
				},
			},
			"application_access_grant": {
				MarkdownDescription: "Application Access Grant Unique Identifier.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"reason": {
				MarkdownDescription: "Reason for revoking or denying approval.",
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

func (t applicationAccessGrantAuthorizationResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationAccessGrantAuthorizationResource{
		provider: provider,
	}, diags
}

type GrantAuthorizationData struct {
	ApplicationAccessGrant types.String `tfsdk:"application_access_grant"`
	Status                 types.String `tfsdk:"status"`
	Reason                 types.String `tfsdk:"reason"`
}

type applicationAccessGrantAuthorizationResource struct {
	provider provider
}

func (r applicationAccessGrantAuthorizationResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data GrantAuthorizationData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Reason.Value == "" {
		data.Reason = types.String{Null: true}
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if data.Status.Value == "Approved" {
		if applicationAccessGrant.Status == "Pending" {
			err := r.provider.client.ApproveGrant(data.ApplicationAccessGrant.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to approve grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to approve grant", "Only a pending grant can be approved")
			return
		}

	} else if data.Status.Value == "Revoked" {
		if applicationAccessGrant.Status == "Approved" {
			err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, data.Reason.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to revoke grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to Revoke grant", "Only approved grant can be revoked")
			return
		}

	} else if data.Status.Value == "Rejected" {
		if applicationAccessGrant.Status == "Pending" {
			err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, data.Reason.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to Deny grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to Deny grant", "Only Pending grant can be denied")
			return
		}
	}

	tflog.Info(ctx, "Saving Grant Authorization resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

}

func (r applicationAccessGrantAuthorizationResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data GrantAuthorizationData

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

	tflog.Info(ctx, "mapping the resource")
	data.ApplicationAccessGrant = types.String{Value: applicationAccessGrant.Uid}
	data.Status = types.String{Value: applicationAccessGrant.Status}
	data.Reason = types.String{Value: applicationAccessGrant.Comment}

	tflog.Info(ctx, "Saving Grant Authorization resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationAccessGrantAuthorizationResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data GrantAuthorizationData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Reason.Value == "" {
		data.Reason = types.String{Null: true}
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.ApplicationAccessGrant.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if data.Status.Value == "Approved" {
		if applicationAccessGrant.Status == "Pending" {
			err := r.provider.client.ApproveGrant(data.ApplicationAccessGrant.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to approve grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to approve grant", "Only a pending grant can be approved")
			return
		}

	} else if data.Status.Value == "Revoked" {
		if applicationAccessGrant.Status == "Approved" {
			err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, data.Reason.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to revoke grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to Revoke grant", "Only approved grant can be revoked")
			return
		}

	} else if data.Status.Value == "Rejected" {
		if applicationAccessGrant.Status == "Pending" {
			err := r.provider.client.RevokeOrDenyGrant(data.ApplicationAccessGrant.Value, data.Reason.Value)
			if err != nil {
				resp.Diagnostics.AddError("Failed to Deny grant", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)
		} else {
			resp.Diagnostics.AddError("Error: Failed to Deny grant", "Only Pending grant can be denied")
			return
		}
	}

	tflog.Info(ctx, "Saving Grant Authorization resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

}

func (r applicationAccessGrantAuthorizationResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	resp.Diagnostics.AddError(
		"Grant Authorization cannot be deleted",
		fmt.Sprint("Application Access Grant Authorization cannot be deleted"),
	)
}

func (r applicationAccessGrantAuthorizationResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("application_access_grant"), req, resp)
}

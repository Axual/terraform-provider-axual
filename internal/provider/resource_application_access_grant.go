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

var _ tfsdk.ResourceType = applicationAccessGrantResourceType{}
var _ tfsdk.Resource = applicationAccessGrantResource{}
var _ tfsdk.ResourceWithImportState = applicationAccessGrantResource{}

type applicationAccessGrantResourceType struct{}

func (t applicationAccessGrantResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {

	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a stream in an environment. Read more: https://docs.axual.io/axual/2022.2/self-service/application-management.html#requesting-stream-access",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "Application Access Grant Unique Identifier",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"status": {
				MarkdownDescription: "Status of Application Access Grant",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"application_access": {
				MarkdownDescription: "Application Access ID to which this Grant belongs",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"application": {
				MarkdownDescription: "Application Unique Identifier",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"stream": {
				MarkdownDescription: "Stream Unique Identifier",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"environment": {
				MarkdownDescription: "Environment Unique Identifier",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"access_type": {
				MarkdownDescription: "Application Access Type. Accepted values: Consumer, Producer",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Consumer", "Producer"}),
				},
			},
		},
	}, nil
}

func (t applicationAccessGrantResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationAccessGrantResource{
		provider: provider,
	}, diags
}

type applicationAccessGrantData struct {
	Id                  types.String `tfsdk:"id"`
	ApplicationId       types.String `tfsdk:"application"`
	StreamId            types.String `tfsdk:"stream"`
	EnvironmentId       types.String `tfsdk:"environment"`
	Status              types.String `tfsdk:"status"`
	AccessType          types.String `tfsdk:"access_type"`
	ApplicationAccessId types.String `tfsdk:"application_access"`
}

type applicationAccessGrantResource struct {
	provider provider
}

func (r applicationAccessGrantResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data applicationAccessGrantData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// get or create application access
	applicationAccess, err := r.provider.client.GetOrCreateApplicationAccess(data.ApplicationId.Value, data.StreamId.Value, data.AccessType.Value)
	if err != nil {
		resp.Diagnostics.AddError("Error getting or creating Application Access", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	grantExists := false
	// get application access grant matching (stream, application, accessType)[ApplicationAccess] and Environment
	for _, grant := range applicationAccess.Embedded.Grants {
		if grant.Environment.Uid == data.EnvironmentId.Value && applicationAccess.AccessType == data.AccessType.Value && grant.Status == "Approved" {
			grantExists = true
			data.Id = types.String{Value: grant.Uid}
			data.Status = types.String{Value: grant.Status}
			data.ApplicationAccessId = types.String{Value: applicationAccess.Uid}
			break
		}
	}
	// if none, make one, if it exists, return it.
	if grantExists == false {
		accessGrantId, err2 := r.provider.client.CreateApplicationAccessGrant(applicationAccess.Uid, data.EnvironmentId.Value)
		if err2 != nil {
			resp.Diagnostics.AddError("Error creating Application Access Grant", fmt.Sprintf("Error message: %s", err2.Error()))
			return
		}

		accessGrant, err3 := r.provider.client.GetApplicationAccessGrant(accessGrantId)
		if err3 != nil {
			resp.Diagnostics.AddError("Error getting Application Access Grant", fmt.Sprintf("Error message: %s", err3.Error()))
			return
		}
		data.Id = types.String{Value: accessGrant.Uid}
		data.Status = types.String{Value: accessGrant.Status}
		data.ApplicationAccessId = types.String{Value: applicationAccess.Uid}
	}

	tflog.Trace(ctx, "Created Application Access Grant resource")
	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationAccessGrantResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("GET request error with application access grant resource(GetApplicationAccessGrant)", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	tflog.Info(ctx, "mapping the resource")
	data.Id = types.String{Value: applicationAccessGrant.Uid}
	data.Status = types.String{Value: applicationAccessGrant.Status}

	applicationAccesses, err := r.provider.client.SearchApplicationAccessByStreamAndApplication(data.StreamId.Value, data.ApplicationId.Value)
	if err != nil {
		resp.Diagnostics.AddError("GET request error with application access grant resource(SearchApplicationAccessByStreamAndApplication)", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	for _, applicationAccess := range applicationAccesses.Embedded.ApplicationAccess {
		if applicationAccess.AccessType == data.AccessType.Value {
			data.ApplicationAccessId = types.String{Value: applicationAccess.Uid}
			break
		}
	}

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationAccessGrantResource) Update(_ context.Context, _ tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant cannot be updated",
		fmt.Sprint("To update Application Access Grant resource please delete and create new axual_application_access_grant resource"),
	)
}

func (r applicationAccessGrantResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteApplicationAccessGrant(data.ApplicationAccessId.Value, data.EnvironmentId.Value)
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error with application access grant resource(DeleteApplicationAccessGrant)", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func (r applicationAccessGrantResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
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
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a topic in an environment. Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#requesting-topic-access",
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
			"application": {
				MarkdownDescription: "Application Unique Identifier",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"topic": {
				MarkdownDescription: "Topic Unique Identifier",
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
				MarkdownDescription: "Application Access Type. Accepted values: CONSUMER, PRODUCER",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"CONSUMER", "PRODUCER"}),
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
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application"`
	TopicId      types.String `tfsdk:"topic"`
	EnvironmentId types.String `tfsdk:"environment"`
	Status        types.String `tfsdk:"status"`
	AccessType    types.String `tfsdk:"access_type"`
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

	applicationAccessGrantRequestData := webclient.ApplicationAccessGrantRequest{
		EnvironmentId: data.EnvironmentId.Value,
		TopicId:      data.TopicId.Value,
		ApplicationId: data.ApplicationId.Value,
		AccessType:    data.AccessType.Value,
	}

	ApplicationAccessGrant, err := r.provider.client.CreateApplicationAccessGrant(applicationAccessGrantRequestData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	data.Id = types.String{Value: ApplicationAccessGrant.Uid}
	data.Status = types.String{Value: ApplicationAccessGrant.Status}
	data.TopicId = types.String{Value: data.TopicId.Value}
	data.EnvironmentId = types.String{Value: ApplicationAccessGrant.Environment.Id}
	data.ApplicationId = types.String{Value: data.ApplicationId.Value}

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
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application Access Grant not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	data.Id = types.String{Value: applicationAccessGrant.Uid}
	data.Status = types.String{Value: applicationAccessGrant.Status}
	data.TopicId = types.String{Value: data.TopicId.Value}
	data.EnvironmentId = types.String{Value: applicationAccessGrant.Embedded.Environment.Uid}
	data.ApplicationId = types.String{Value: data.ApplicationId.Value}

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationAccessGrantResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant cannot be updated",
		"If you would like to cancel this request, delete the resource. This is only possible if the request is still pending.",
	)
}

func (r applicationAccessGrantResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Links.Cancel.Href != "" {
		err1 := r.provider.client.CancelGrant(data.Id.Value)
		if err1 != nil {
			resp.Diagnostics.AddError("Unable to cancel Application Access Grant", fmt.Sprintf("Error message: %s", err1))
			return
		}
		return
	}

	if applicationAccessGrant.Status == "Approved" {
		resp.Diagnostics.AddError(
			"Application Access Grant cannot be cancelled",
			fmt.Sprintf(
				"Please Revoke this grant before attempting to delete it.\nCurrent Status of the grant: %s",
				applicationAccessGrant.Status))
		return
	}

}

func (r applicationAccessGrantResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

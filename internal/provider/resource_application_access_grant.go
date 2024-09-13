package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"time"
)

var _ resource.Resource = &applicationAccessGrantResource{}
var _ resource.ResourceWithImportState = &applicationAccessGrantResource{}

func NewApplicationAccessGrantResource(provider AxualProvider) resource.Resource {
	return &applicationAccessGrantResource{
		provider: provider,
	}
}

type applicationAccessGrantResource struct {
	provider AxualProvider
}
type applicationAccessGrantData struct {
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application"`
	TopicId       types.String `tfsdk:"topic"`
	EnvironmentId types.String `tfsdk:"environment"`
	Status        types.String `tfsdk:"status"`
	AccessType    types.String `tfsdk:"access_type"`
}

func (r *applicationAccessGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant"
}

func (r *applicationAccessGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a topic in an environment. Read more: https://docs.axual.io/axual/2024.2/self-service/application-management.html#requesting-topic-access",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Application Access Grant Unique Identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of Application Access Grant",
				Computed:            true,
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "Application Unique Identifier",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"topic": schema.StringAttribute{
				MarkdownDescription: "Topic Unique Identifier",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment Unique Identifier",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_type": schema.StringAttribute{
				MarkdownDescription: "Application Access Type. Accepted values: CONSUMER, PRODUCER",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("CONSUMER", "PRODUCER"),
				},
			},
		},
	}
}

func (r *applicationAccessGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationAccessGrantData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrantRequestData := webclient.ApplicationAccessGrantRequest{
		EnvironmentId: data.EnvironmentId.ValueString(),
		StreamId:      data.TopicId.ValueString(),
		ApplicationId: data.ApplicationId.ValueString(),
		AccessType:    data.AccessType.ValueString(),
	}

	ApplicationAccessGrant, err := r.provider.client.CreateApplicationAccessGrant(applicationAccessGrantRequestData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	data.Id = types.StringValue(ApplicationAccessGrant.Uid)
	data.Status = types.StringValue(ApplicationAccessGrant.Status)
	data.TopicId = types.StringValue(data.TopicId.ValueString())
	data.EnvironmentId = types.StringValue(ApplicationAccessGrant.Environment.Id)
	data.ApplicationId = types.StringValue(data.ApplicationId.ValueString())

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationAccessGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application Access Grant not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	data.Id = types.StringValue(applicationAccessGrant.Uid)
	data.Status = types.StringValue(applicationAccessGrant.Status)
	data.TopicId = types.StringValue(data.TopicId.ValueString())
	data.EnvironmentId = types.StringValue(applicationAccessGrant.Embedded.Environment.Uid)
	data.ApplicationId = types.StringValue(data.ApplicationId.ValueString())

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationAccessGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant cannot be updated",
		"If you would like to cancel this request, delete the resource. This is only possible if the request is still pending.",
	)
}

func (r *applicationAccessGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	time.Sleep(10 * time.Second) // Just retry does not work
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if applicationAccessGrant.Links.Cancel.Href != "" {
		// Retry logic for cancelling the grant
		err1 := Retry(3, 3*time.Second, func() error {
			return r.provider.client.CancelGrant(data.Id.ValueString())
		})
		if err1 != nil {
			resp.Diagnostics.AddError("Unable to cancel Application Access Grant", fmt.Sprintf("Error message after retries: %s", err1))
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

func (r *applicationAccessGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

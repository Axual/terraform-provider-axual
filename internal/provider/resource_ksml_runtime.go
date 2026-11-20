package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Lengths the API enforces on a KSML runtime.
const (
	ksmlRuntimeNameMinLength        = 3
	ksmlRuntimeNameMaxLength        = 50
	ksmlRuntimeDescriptionMaxLength = 255
	ksmlRuntimeUrlMaxLength         = 255
)

var _ resource.Resource = &ksmlRuntimeResource{}
var _ resource.ResourceWithImportState = &ksmlRuntimeResource{}

func NewKsmlRuntimeResource(provider AxualProvider) resource.Resource {
	return &ksmlRuntimeResource{
		provider: provider,
	}
}

type ksmlRuntimeResource struct {
	provider AxualProvider
}

type KsmlRuntimeResourceData struct {
	Id               types.String `tfsdk:"id"`
	InstanceId       types.String `tfsdk:"instance_id"`
	ClusterId        types.String `tfsdk:"cluster_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Url              types.String `tfsdk:"url"`
	SchemaRegistries types.List   `tfsdk:"schema_registries"`
	Owners           types.String `tfsdk:"owners"`
	AuthorizedGroups types.Set    `tfsdk:"authorized_groups"`
}

func (r *ksmlRuntimeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ksml_runtime"
}

func (r *ksmlRuntimeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A KSML runtime registers a KSML provisioner on an Instance-Cluster, so KSML applications can be deployed to it.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Instance this KSML runtime belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Cluster this KSML runtime belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the KSML runtime. Must be between 3 and 50 characters, and unique within your tenant, ignoring case. Platform Manager does not allow renaming, so changing the name destroys and re-creates the KSML runtime.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(ksmlRuntimeNameMinLength, ksmlRuntimeNameMaxLength),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The id of the Group that owns the KSML runtime, for example `data.axual_group.platform_team.id`. Members of the owner Group can edit the KSML runtime, but owning it does not allow deploying to it: add the Group to `authorized_groups` for that. Only a Tenant Admin or a Platform Admin can change the owner.",
				Required:            true,
			},
			"authorized_groups": schema.SetAttribute{
				MarkdownDescription: "The ids of the Groups that may deploy KSML applications to the KSML runtime. Leave it out to keep the Groups the KSML runtime has; `authorized_groups = []` removes all of them. **A KSML runtime without authorized Groups accepts no deployments, from the owner Group either.** A Group cannot be removed while it has a running KSML deployment on this KSML runtime.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short description of the KSML runtime. Must be at most 255 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(ksmlRuntimeDescriptionMaxLength),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the KSML provisioner, e.g. `https://ksml.example.com`. Must be at most 255 characters.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(ksmlRuntimeUrlMaxLength),
				},
			},
			"schema_registries": schema.ListNestedAttribute{
				MarkdownDescription: "Schema registries the KSML applications on this KSML runtime read schemas from.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Schema registry type: `CONFLUENT` or `APICURIO`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("CONFLUENT", "APICURIO"),
							},
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "Base URL of the schema registry API, e.g. `https://apicurio.example.com/apis/ccompat/v7` for Confluent compatibility.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(ksmlRuntimeUrlMaxLength),
							},
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "KSML runtime unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ksmlRuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KsmlRuntimeResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ksmlRuntimeRequest := createKsmlRuntimeRequestFromData(&data)

	ksmlRuntime, err := r.provider.client.CreateKsmlRuntime(data.InstanceId.ValueString(), data.ClusterId.ValueString(), ksmlRuntimeRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for KSML runtime resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapKsmlRuntimeResponseToData(&data, ksmlRuntime)
	tflog.Info(ctx, "created KSML runtime")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ksmlRuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KsmlRuntimeResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ksmlRuntime, err := r.provider.client.GetKsmlRuntime(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString())
	if err != nil {
		if isGone(err) {
			tflog.Warn(ctx, fmt.Sprintf("KSML runtime not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read KSML runtime, got error: %s", err))
		}
		return
	}

	mapKsmlRuntimeResponseToData(&data, ksmlRuntime)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ksmlRuntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KsmlRuntimeResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ksmlRuntimeRequest := createKsmlRuntimeRequestFromData(&data)

	ksmlRuntime, err := r.provider.client.UpdateKsmlRuntime(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString(), ksmlRuntimeRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update KSML runtime, got error: %s", err))
		return
	}

	mapKsmlRuntimeResponseToData(&data, ksmlRuntime)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ksmlRuntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KsmlRuntimeResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := ignoreGone(r.provider.client.DeleteKsmlRuntime(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete KSML runtime, got error: %s", err))
		return
	}
}

// ImportState accepts an id of the form "instance_id/cluster_id/ksml_runtime_id": the API finds a
// KSML runtime through the instance and cluster it is registered on.
func (r *ksmlRuntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id/cluster_id/ksml_runtime_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func createKsmlRuntimeRequestFromData(data *KsmlRuntimeResourceData) webclient.KsmlRuntimeRequest {
	// A removed description is sent as an explicit null, which is what the PATCH needs to clear it.
	var description *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		value := data.Description.ValueString()
		description = &value
	}

	return webclient.KsmlRuntimeRequest{
		OwnerGroupId:       data.Owners.ValueString(),
		AuthorizedGroupIds: authorizedGroupIdsForRequest(data.AuthorizedGroups),
		Name:               data.Name.ValueString(),
		Description:        description,
		Url:                data.Url.ValueString(),
		SchemaRegistries:   schemaRegistriesForRequest(data.SchemaRegistries),
	}
}

func mapKsmlRuntimeResponseToData(data *KsmlRuntimeResourceData, ksmlRuntime *webclient.KsmlRuntimeResponse) {
	data.Id = types.StringValue(ksmlRuntime.Uid)
	data.Name = types.StringValue(ksmlRuntime.Name)
	data.Url = types.StringValue(ksmlRuntime.Url)
	data.SchemaRegistries = schemaRegistriesToList(ksmlRuntime.SchemaRegistries)
	data.AuthorizedGroups = authorizedGroupsToSet(ksmlRuntime.AuthorizedGroups)
	// A runtime that was a KSML Provisioner before Platform Manager had KSML runtimes has no owner.
	data.Owners = stringOrNull(ksmlRuntime.OwnerGroupId)
	data.Description = stringOrNull(ksmlRuntime.Description)
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Lengths the API enforces on a Flink Cluster, from FlinkClusterConstants and CoreConstants:
// name, workspace, namespace and deployment target share DEFAULT_ENTITY_NAME_MAX_LENGTH, the url
// uses DEFAULT_ENTITY_URL_MAX_LENGTH, and the description is capped at 255 on both the entity and
// the request DTO.
const (
	flinkClusterNameMinLength        = 3
	flinkClusterNameMaxLength        = 50
	flinkClusterDescriptionMaxLength = 255
	flinkClusterUrlMaxLength         = 255
)

var _ resource.Resource = &flinkClusterResource{}
var _ resource.ResourceWithImportState = &flinkClusterResource{}

func NewFlinkClusterResource(provider AxualProvider) resource.Resource {
	return &flinkClusterResource{
		provider: provider,
	}
}

type flinkClusterResource struct {
	provider AxualProvider
}

type FlinkClusterResourceData struct {
	Id               types.String `tfsdk:"id"`
	InstanceId       types.String `tfsdk:"instance_id"`
	ClusterId        types.String `tfsdk:"cluster_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Url              types.String `tfsdk:"url"`
	Workspace        types.String `tfsdk:"workspace"`
	Namespace        types.String `tfsdk:"namespace"`
	DeploymentTarget types.String `tfsdk:"deployment_target"`
	ApiToken         types.String `tfsdk:"api_token"`
	SchemaRegistries types.List   `tfsdk:"schema_registries"`
}

func (r *flinkClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flink_cluster"
}

func (r *flinkClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Flink Cluster registers a Ververica Platform namespace on an Instance-Cluster, so Flink SQL applications can be deployed to it.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Instance this Flink Cluster belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Cluster this Flink Cluster belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the Flink Cluster.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(flinkClusterNameMinLength, flinkClusterNameMaxLength),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short description of the Flink Cluster.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(flinkClusterDescriptionMaxLength),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Ververica Platform base URL, e.g. `https://vvp.internal`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(flinkClusterUrlMaxLength),
				},
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Ververica workspace name, e.g. `defaultworkspace`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(flinkClusterNameMaxLength),
				},
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Ververica namespace name, e.g. `default`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(flinkClusterNameMaxLength),
				},
			},
			"deployment_target": schema.StringAttribute{
				MarkdownDescription: "Ververica deployment target name, e.g. `default-target`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(flinkClusterNameMaxLength),
				},
			},
			"schema_registries": schema.ListNestedAttribute{
				MarkdownDescription: "Schema registries the Flink Cluster reads schemas from. Required for Flink SQL jobs over AVRO topics: the platform injects the url into the generated table DDL. When more than one is configured, a job using `'value.format' = 'avro-confluent'` uses the first Confluent-type registry in the list.",
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
								stringvalidator.LengthAtMost(flinkClusterUrlMaxLength),
							},
						},
					},
				},
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "Ververica namespace-scoped API token. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands.",
				Required:            true,
				Sensitive:           true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Flink Cluster unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *flinkClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FlinkClusterResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	flinkClusterRequest := createFlinkClusterRequestFromData(&data)

	flinkCluster, err := r.provider.client.CreateFlinkCluster(data.InstanceId.ValueString(), data.ClusterId.ValueString(), flinkClusterRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for flink cluster resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapFlinkClusterResponseToData(&data, flinkCluster)
	tflog.Info(ctx, "created Flink Cluster")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *flinkClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlinkClusterResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	flinkCluster, err := r.provider.client.GetFlinkCluster(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Flink Cluster not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Flink Cluster, got error: %s", err))
		}
		return
	}

	mapFlinkClusterResponseToData(&data, flinkCluster)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *flinkClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FlinkClusterResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	flinkClusterRequest := createFlinkClusterRequestFromData(&data)

	flinkCluster, err := r.provider.client.UpdateFlinkCluster(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString(), flinkClusterRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Flink Cluster, got error: %s", err))
		return
	}

	mapFlinkClusterResponseToData(&data, flinkCluster)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *flinkClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FlinkClusterResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteFlinkCluster(data.InstanceId.ValueString(), data.ClusterId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Flink Cluster, got error: %s", err))
		return
	}
}

// ImportState accepts an id of the form "instance_id/cluster_id/flink_cluster_id",
// since the Flink Cluster's own uid is not unique without the instance/cluster it lives on.
func (r *flinkClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id/cluster_id/flink_cluster_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func createFlinkClusterRequestFromData(data *FlinkClusterResourceData) webclient.FlinkClusterRequest {
	// A removed description is sent as an explicit null, which is what the PATCH needs to clear it.
	var description *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		value := data.Description.ValueString()
		description = &value
	}

	// The same holds for the schema registries: an omitted key leaves the stored list in place,
	// so a removed block is sent as an explicit null.
	var schemaRegistries *[]webclient.SchemaRegistryConfig
	if !data.SchemaRegistries.IsNull() && !data.SchemaRegistries.IsUnknown() {
		registries := make([]webclient.SchemaRegistryConfig, 0, len(data.SchemaRegistries.Elements()))
		for _, element := range data.SchemaRegistries.Elements() {
			object, ok := element.(types.Object)
			if !ok {
				continue
			}
			attributes := object.Attributes()
			registry := webclient.SchemaRegistryConfig{}
			if value, ok := attributes["type"].(types.String); ok {
				registry.Type = value.ValueString()
			}
			if value, ok := attributes["url"].(types.String); ok {
				registry.Url = value.ValueString()
			}
			registries = append(registries, registry)
		}
		schemaRegistries = &registries
	}

	return webclient.FlinkClusterRequest{
		Name:             data.Name.ValueString(),
		Description:      description,
		Url:              data.Url.ValueString(),
		Workspace:        data.Workspace.ValueString(),
		Namespace:        data.Namespace.ValueString(),
		DeploymentTarget: data.DeploymentTarget.ValueString(),
		ApiToken:         data.ApiToken.ValueString(),
		SchemaRegistries: schemaRegistries,
	}
}

// schemaRegistryObjectType mirrors the nested object of the `schema_registries` attribute.
func schemaRegistryObjectType() attr.TypeWithAttributeTypes {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type": types.StringType,
		"url":  types.StringType,
	}}
}

// mapSchemaRegistriesToData writes the API's list into state, keeping an unset attribute null: the
// API answers with an empty list for a cluster that has none, which is not the same as `[]`.
func mapSchemaRegistriesToData(data *FlinkClusterResourceData, registries []webclient.SchemaRegistryConfig) {
	objectType := schemaRegistryObjectType()
	if len(registries) == 0 {
		data.SchemaRegistries = types.ListNull(objectType)
		return
	}

	elements := make([]attr.Value, 0, len(registries))
	for _, registry := range registries {
		object, diags := types.ObjectValue(objectType.AttributeTypes(), map[string]attr.Value{
			"type": types.StringValue(registry.Type),
			"url":  types.StringValue(registry.Url),
		})
		if diags.HasError() {
			continue
		}
		elements = append(elements, object)
	}

	list, diags := types.ListValue(objectType, elements)
	if diags.HasError() {
		data.SchemaRegistries = types.ListNull(objectType)
		return
	}
	data.SchemaRegistries = list
}

func mapFlinkClusterResponseToData(data *FlinkClusterResourceData, flinkCluster *webclient.FlinkClusterResponse) {
	data.Id = types.StringValue(flinkCluster.Uid)
	data.Name = types.StringValue(flinkCluster.Name)
	data.Url = types.StringValue(flinkCluster.Url)
	data.Workspace = types.StringValue(flinkCluster.Workspace)
	data.Namespace = types.StringValue(flinkCluster.Namespace)
	data.DeploymentTarget = types.StringValue(flinkCluster.DeploymentTarget)
	mapSchemaRegistriesToData(data, flinkCluster.SchemaRegistries)

	if flinkCluster.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(flinkCluster.Description)
	}
}

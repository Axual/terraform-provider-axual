package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ datasource.DataSource                     = &connectClusterDataSource{}
	_ datasource.DataSourceWithConfigValidators = &connectClusterDataSource{}
)

func NewConnectClusterDataSource(provider AxualProvider) datasource.DataSource {
	return &connectClusterDataSource{
		provider: provider,
	}
}

type connectClusterDataSource struct {
	provider AxualProvider
}

type connectClusterDataSourceData struct {
	InstanceId       types.String `tfsdk:"instance_id"`
	ClusterId        types.String `tfsdk:"cluster_id"`
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ConnectUrl       types.String `tfsdk:"connect_url"`
	AuthMethod       types.String `tfsdk:"auth_method"`
	LogViewerUrl     types.String `tfsdk:"log_viewer_url"`
	OwnerGroupId     types.String `tfsdk:"owner_group_id"`
	OwnerGroupName   types.String `tfsdk:"owner_group_name"`
	AuthorizedGroups types.List   `tfsdk:"authorized_group_ids"`
}

func (d *connectClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connect_cluster"
}

func (d *connectClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Kafka Connect cluster already registered on an Instance-Cluster, so its id can be used as an `axual_application_deployment`'s `target_id`. Registering, changing or removing a Kafka Connect cluster is an administrator action outside Terraform; this data source is read-only. Needs Platform Manager 16.0.0 or later.",

		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Instance the Kafka Connect cluster belongs to.",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Cluster the Kafka Connect cluster belongs to.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The Kafka Connect cluster's unique identifier. Provide this or `name`, not both.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The Kafka Connect cluster's name. Provide this or `id`, not both. Looking up by name reads every page of the cluster list, since the API has no server-side name filter for this resource.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short description of the Kafka Connect cluster.",
				Computed:            true,
			},
			"connect_url": schema.StringAttribute{
				MarkdownDescription: "The Kafka Connect cluster's REST API base URL.",
				Computed:            true,
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "The Kafka authentication method every connector on this cluster must use: `MTLS` or `SASL_SCRAM`. Not the same as how Platform Manager itself authenticates to the Connect REST API. A Connector `axual_application_deployment` targeting a `SASL_SCRAM` cluster needs an `axual_application_credential` supporting `SCRAM_SHA_512`; one targeting an `MTLS` cluster needs an active `axual_application_principal`, same as legacy Axual Connect.",
				Computed:            true,
			},
			"log_viewer_url": schema.StringAttribute{
				MarkdownDescription: "Address of the log provisioner for this cluster, if one is configured.",
				Computed:            true,
			},
			"owner_group_id": schema.StringAttribute{
				MarkdownDescription: "Uid of the group that owns this Kafka Connect cluster registration.",
				Computed:            true,
			},
			"owner_group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the group that owns this Kafka Connect cluster registration.",
				Computed:            true,
			},
			"authorized_group_ids": schema.ListAttribute{
				MarkdownDescription: "Uids of the groups authorized to deploy connectors to this cluster.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *connectClusterDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *connectClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data connectClusterDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	instanceId := data.InstanceId.ValueString()
	clusterId := data.ClusterId.ValueString()

	id := data.Id.ValueString()
	if id == "" {
		found, err := d.provider.client.FindConnectClusterByName(instanceId, clusterId, data.Name.ValueString())
		if err != nil {
			if errors.Is(err, webclient.NotFoundError) {
				resp.Diagnostics.AddError(
					"Resource Not Found",
					fmt.Sprintf("No Kafka Connect cluster found with name '%s' on instance '%s' cluster '%s'.", data.Name.ValueString(), instanceId, clusterId),
				)
				return
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find Kafka Connect cluster by name, got error: %s", err))
			return
		}
		id = found.Id
	}

	connectCluster, err := d.provider.client.GetConnectCluster(instanceId, clusterId, id)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError("Resource Not Found", fmt.Sprintf("No Kafka Connect cluster found with id '%s'.", id))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Kafka Connect cluster, got error: %s", err))
		return
	}

	mapConnectClusterDataSourceResponseToData(&data, connectCluster)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapConnectClusterDataSourceResponseToData(data *connectClusterDataSourceData, connectCluster *webclient.ConnectClusterResponse) {
	data.Id = types.StringValue(connectCluster.Id)
	data.Name = types.StringValue(connectCluster.Name)
	data.ConnectUrl = types.StringValue(connectCluster.ConnectUrl)
	data.AuthMethod = types.StringValue(connectCluster.AuthMethod)
	data.OwnerGroupId = types.StringValue(connectCluster.OwnerGroupId)
	data.OwnerGroupName = types.StringValue(connectCluster.OwnerGroupName)

	data.Description = stringOrNull(connectCluster.Description)
	data.LogViewerUrl = stringOrNull(connectCluster.LogViewerUrl)

	groupIds := make([]attr.Value, 0, len(connectCluster.AuthorizedGroups))
	for _, group := range connectCluster.AuthorizedGroups {
		groupIds = append(groupIds, types.StringValue(group.Id))
	}
	list, listDiags := types.ListValue(types.StringType, groupIds)
	if listDiags.HasError() {
		data.AuthorizedGroups = types.ListNull(types.StringType)
	} else {
		data.AuthorizedGroups = list
	}
}

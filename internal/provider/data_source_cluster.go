package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &clusterDataSource{}

func NewClusterDataSource(provider AxualProvider) datasource.DataSource {
	return &clusterDataSource{
		provider: provider,
	}
}

type clusterDataSource struct {
	provider AxualProvider
}

type clusterDataSourceData struct {
	Id                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	Location                      types.String `tfsdk:"location"`
	ProviderType                  types.String `tfsdk:"provider_type"`
	Shared                        types.Bool   `tfsdk:"shared"`
	TopicPattern                  types.String `tfsdk:"topic_pattern"`
	GroupPattern                  types.String `tfsdk:"group_pattern"`
	TransactionalPattern          types.String `tfsdk:"transactional_pattern"`
	KafkaVersion                  types.String `tfsdk:"kafka_version"`
	BootstrapUrlsByAuthentication types.Map    `tfsdk:"bootstrap_urls_by_authentication"`
}

func (d *clusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *clusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Cluster by name, for example to add an existing Cluster to an `axual_instance`. Broker credentials are not exposed.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The exact name of the Cluster.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(clusterNameMinLength, clusterNameMaxLength),
				},
			},
			"id":                    schema.StringAttribute{Computed: true, MarkdownDescription: "Cluster unique identifier"},
			"description":           schema.StringAttribute{Computed: true, MarkdownDescription: "The description of the Cluster."},
			"location":              schema.StringAttribute{Computed: true, MarkdownDescription: "Where the Cluster runs."},
			"provider_type":         schema.StringAttribute{Computed: true, MarkdownDescription: "`apache-kafka` or `strimzi`."},
			"shared":                schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the Cluster is shared by all tenants."},
			"topic_pattern":         schema.StringAttribute{Computed: true, MarkdownDescription: "How a topic name is made on the Cluster."},
			"group_pattern":         schema.StringAttribute{Computed: true, MarkdownDescription: "How a consumer group id is made on the Cluster."},
			"transactional_pattern": schema.StringAttribute{Computed: true, MarkdownDescription: "How a transactional id is made on the Cluster."},
			"kafka_version":         schema.StringAttribute{Computed: true, MarkdownDescription: "The Kafka version of the Cluster."},
			"bootstrap_urls_by_authentication": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The bootstrap servers applications use, per authentication method.",
			},
		},
	}
}

func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clusterDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := findClusterIDByName(d.provider.client, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	cluster, err := d.provider.client.GetCluster(id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Cluster %q, got error: %s", data.Name.ValueString(), err))
		return
	}

	data.Id = types.StringValue(cluster.Uid)
	data.Name = types.StringValue(cluster.Name)
	data.Description = stringOrNull(cluster.Description)
	data.Location = stringOrNull(cluster.Location)
	data.ProviderType = stringOrNull(cluster.ProviderType)
	data.Shared = types.BoolValue(cluster.SharedCluster)
	data.TopicPattern = stringOrNull(cluster.TopicPattern)
	data.GroupPattern = stringOrNull(cluster.GroupPattern)
	data.TransactionalPattern = stringOrNull(cluster.TransactionalPattern)
	data.KafkaVersion = stringOrNull(cluster.KafkaVersion)
	data.BootstrapUrlsByAuthentication = stringMapToData(cluster.BootstrapUrlsByAuthentication)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

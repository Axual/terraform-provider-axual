package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = topicDataSourceType{}
var _ tfsdk.DataSource = topicDataSource{}

type topicDataSourceType struct{}

func (t topicDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A topic represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2024.1/self-service/topic-management.html",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "The name of the topic. This must be in the format string-string (Needs to contain exactly one dash). The topic name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the topic.",
				Optional:            true,
				Type:                types.StringType,
				Computed:            true,
			},
			"key_type": {
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2024.1/self-service/topic-management.html#key-type",
				Type:                types.StringType,
				Computed:            true,
			},
			"value_type": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2024.1/self-service/topic-management.html#value-type",

				Type:     types.StringType,
				Computed: true,
			},
			"owners": {
				MarkdownDescription: "The team owning this topic. Read more: https://docs.axual.io/axual/2024.1/self-service/topic-management.html#topic-owner",
				Type:                types.StringType,
				Computed:            true,
			},
			"retention_policy": {
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2024.1/self-service/topic-management.html#retention-policy",
				Type:                types.StringType,
				Computed:            true,
			},
			"properties": {
				MarkdownDescription: "Advanced (Kafka) properties for a topic in a given environment. Read more: https://docs.axual.io/axual/2024.1/self-service/advanced-features.html#configuring-topic-properties",
				Computed:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Topic unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t topicDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return topicDataSource{
		provider: provider,
	}, diags
}

type topicDataSourceData struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	KeyType         types.String `tfsdk:"key_type"`
	ValueType       types.String `tfsdk:"value_type"`
	Owners          types.String `tfsdk:"owners"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Id              types.String `tfsdk:"id"`
	Properties      types.Map    `tfsdk:"properties"`
}

type topicDataSource struct {
	provider provider
}

func (d topicDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data topicDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicByName, err := d.provider.client.GetTopicByName(data.Name.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic by name, got error: %s", err))
		return
	}

	topic, err := d.provider.client.GetTopic(topicByName.Embedded.Topics[0].Uid)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic, got error: %s", err))
		return
	}
	mapTopicDataSourceResponseToData(ctx, &data, topic)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapTopicDataSourceResponseToData(ctx context.Context, data *topicDataSourceData, topic *webclient.TopicResponse) {

	data.Id = types.String{Value: topic.Uid}
	data.Name = types.String{Value: topic.Name}
	data.KeyType = types.String{Value: topic.KeyType}
	data.ValueType = types.String{Value: topic.ValueType}
	data.Owners = types.String{Value: topic.Embedded.Owners.Uid}
	data.RetentionPolicy = types.String{Value: topic.RetentionPolicy}

	properties := make(map[string]attr.Value)
	for key, value := range topic.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}

	// optional fields
	if topic.Description == nil || len(topic.Description.(string)) == 0 {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: topic.Description.(string)}
	}
}

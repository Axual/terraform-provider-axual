package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &topicDataSource{}

func NewTopicDataSource(provider AxualProvider) datasource.DataSource {
	return &topicDataSource{
		provider: provider,
	}
}

type topicDataSource struct {
	provider AxualProvider
}

type topicDataSourceData struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	KeyType         types.String `tfsdk:"key_type"`
	KeySchema       types.String `tfsdk:"key_schema"`
	ValueType       types.String `tfsdk:"value_type"`
	ValueSchema     types.String `tfsdk:"value_schema"`
	Owners          types.String `tfsdk:"owners"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Id              types.String `tfsdk:"id"`
	Properties      types.Map    `tfsdk:"properties"`
}

func (d *topicDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (d *topicDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A topic represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2024.2/self-service/topic-management.html",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the topic. Can only contain letters, numbers, dots, dashes and underscores and cannot begin with an underscore, dot or dash, but can't start with underscore, dot or dash. The topic name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 180),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`), "can only contain letters, numbers, dots, dashes and underscores and cannot begin with an underscore, dot or dash"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A text describing the purpose of the topic.",
				Computed:            true,
			},
			"key_type": schema.StringAttribute{
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2024.2/self-service/topic-management.html#key-type",
				Computed:            true,
			},
			"value_type": schema.StringAttribute{
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2024.2/self-service/topic-management.html#value-type",
				Computed:            true,
			},
			"key_schema": schema.StringAttribute{
				MarkdownDescription: "The key schema UID if `key_type` is 'AVRO'.",
				Computed:            true,
			},
			"value_schema": schema.StringAttribute{
				MarkdownDescription: "The value schema UID if `value_type` is 'AVRO'.",
				Computed:            true,
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The team owning this topic. Read more: https://docs.axual.io/axual/2024.2/self-service/topic-management.html#topic-owner",
				Computed:            true,
			},
			"retention_policy": schema.StringAttribute{
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2024.2/self-service/topic-management.html#retention-policy",
				Computed:            true,
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "Advanced (Kafka) properties for a topic in a given environment. Read more: https://docs.axual.io/axual/2024.2/self-service/advanced-features.html#configuring-topic-properties",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Topic unique identifier",
			},
		},
	}
}

func (d *topicDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data topicDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicByName, err := d.provider.client.GetTopicByName(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic by name, got error: %s", err))
		return
	}

	// Check if Embedded or topic is nil or empty
	if len(topicByName.Embedded.Topics) == 0 {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("No Topic resources found with name '%s'.", data.Name.ValueString()),
		)
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

	data.Id = types.StringValue(topic.Uid)
	data.Name = types.StringValue(topic.Name)
	data.KeyType = types.StringValue(topic.KeyType)
	data.ValueType = types.StringValue(topic.ValueType)
	data.Owners = types.StringValue(topic.Embedded.Owners.Uid)
	data.RetentionPolicy = types.StringValue(topic.RetentionPolicy)

	properties := make(map[string]attr.Value)
	for key, value := range topic.Properties {
		if value != nil {
			properties[key] = types.StringValue(value.(string))
		}
	}

	mapValue, diags := types.MapValue(types.StringType, properties)

	if diags.HasError() {
		tflog.Error(ctx, "Error creating members slice when mapping group response")
	}

	data.Properties = mapValue

	// optional fields
	if topic.Description == nil || len(topic.Description.(string)) == 0 {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(topic.Description.(string))
	}

	// Map key_schema if KeyType is AVRO
	if data.KeyType.ValueString() == "AVRO" {
		if topic.Embedded.KeySchema.Uid != "" {
			data.KeySchema = types.StringValue(topic.Embedded.KeySchema.Uid)
		} else {
			data.KeySchema = types.StringNull()
		}
	} else {
		data.KeySchema = types.StringNull()
	}

	// Map value_schema if ValueType is AVRO
	if data.ValueType.ValueString() == "AVRO" {
		if topic.Embedded.ValueSchema.Uid != "" {
			data.ValueSchema = types.StringValue(topic.Embedded.ValueSchema.Uid)
		} else {
			data.ValueSchema = types.StringNull()
		}
	} else {
		data.ValueSchema = types.StringNull()
	}
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = topicConfigResourceType{}
var _ tfsdk.Resource = topicConfigResource{}
var _ tfsdk.ResourceWithImportState = topicConfigResource{}

type topicConfigResourceType struct{}

func (t topicConfigResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Topic Config resource. Once the Topic has been created, the next step to actually configure the topic for any environment is to configure the topic. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#configuring-a-topic-for-an-environment",

		Attributes: map[string]tfsdk.Attribute{
			"partitions": {
				MarkdownDescription: "The number of partitions define how many consumer instances can be started in parallel on this topic. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#partitions-number",
				Required:            true,
				Type:                types.Int64Type,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"retention_time": {
				MarkdownDescription: "Determine how long the messages should be available on a topic. There should be an agreed value most likely discussed in Intake session with the team supporting Axual Platform. In most cases, it is 7 days. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#retention-time",
				Required:            true,
				Type:                types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					int64validator.AtLeast(1000),
				},
			},
			"topic": {
				MarkdownDescription: "The Topic this topic configuration is associated with",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"environment": {
				MarkdownDescription: "The environment this topic configuration is associated with",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"key_schema_version": {
				MarkdownDescription: "The schema version this topic configuration supports for the key.",
				Optional:            true,
				Type:                types.StringType,
			},
			"value_schema_version": {
				MarkdownDescription: "The schema version this topic configuration supports for the value.",
				Optional:            true,
				Type:                types.StringType,
			},
			"properties": {
				MarkdownDescription: "You can define Kafka properties for your topic here. segment.ms property needs to always be included. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#configuring-a-topic-for-an-environment",
				Required:            true,
				Type:                types.MapType{ElemType: types.StringType},
				Validators: []tfsdk.AttributeValidator{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.OneOf("segment.ms", "retention.bytes", "min.compaction.lag.ms", "max.compaction.lag.ms", "message.timestamp.difference.max.ms", "message.timestamp.type")),
					mapvalidator.ValuesAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "topic config identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t topicConfigResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return topicConfigResource{
		provider: provider,
	}, diags
}

type topicConfigResourceData struct {
	Partitions         types.Int64  `tfsdk:"partitions"`
	RetentionTime      types.Int64  `tfsdk:"retention_time"`
	Topic              types.String `tfsdk:"topic"`
	Environment        types.String `tfsdk:"environment"`
	KeySchemaVersion   types.String `tfsdk:"key_schema_version"`
	ValueSchemaVersion types.String `tfsdk:"value_schema_version"`
	Id                 types.String `tfsdk:"id"`
	Properties         types.Map    `tfsdk:"properties"`
}

type topicConfigResource struct {
	provider provider
}

func (r topicConfigResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data topicConfigResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.ReadTopic(data.Topic.Value)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if !data.KeySchemaVersion.Null {
		if topic.KeyType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Key Schema. Please don't set the KeySchemaVersion: %s", data.KeySchemaVersion.Value))
			return
		} else {
			r.validateSchemaVersionsForCreate(topic.Embedded.KeySchema.Uid, data.KeySchemaVersion.Value, resp)
			if resp.Diagnostics.HasError() {
				return
			}

		}
	}

	if !data.ValueSchemaVersion.Null {
		if topic.ValueType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Value Schema. Please don't set the ValueSchemaVersion: %s", data.ValueSchemaVersion))
			return
		} else {
			r.validateSchemaVersionsForCreate(topic.Embedded.ValueSchema.Uid, data.ValueSchemaVersion.Value, resp)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	topicConfigRequest, err := createTopicConfigRequestFromData(ctx, &data, r)
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	topicConfigRequest.Properties = properties
	tflog.Info(ctx, fmt.Sprintf("Create topic config request %q", topicConfigRequest))
	topicConfig, err := r.provider.client.CreateTopicConfig(topicConfigRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapTopicConfigResponseToData(ctx, &data, topicConfig)
	tflog.Trace(ctx, "Created a topic config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r topicConfigResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data topicConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicConfig, err := r.provider.client.ReadTopicConfig(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Topic config not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic config, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapTopicConfigResponseToData(ctx, &data, topicConfig)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r topicConfigResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data topicConfigResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.ReadTopic(data.Topic.Value)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if !data.KeySchemaVersion.Null {
		if topic.KeyType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Key Schema. Please don't set the KeySchemaVersion: %s", data.KeySchemaVersion.Value))
			return
		} else {
			r.validateSchemaVersionsForUpdate(topic.Embedded.KeySchema.Uid, data.KeySchemaVersion.Value, resp)
			if resp.Diagnostics.HasError() {
				return
			}

		}
	}

	if !data.ValueSchemaVersion.Null {
		if topic.ValueType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Value Schema. Please don't set the ValueSchemaVersion: %s", data.ValueSchemaVersion))
			return
		} else {
			r.validateSchemaVersionsForUpdate(topic.Embedded.ValueSchema.Uid, data.ValueSchemaVersion.Value, resp)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	topicConfigRequest, err := createTopicConfigRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	var oldPropertiesState map[string]string
	req.State.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("properties"), &oldPropertiesState)

	properties := make(map[string]interface{})

	for key, _ := range oldPropertiesState {
		properties[key] = nil
	}

	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}

	topicConfigRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update topic config request %q", topicConfigRequest))
	topicConfig, err := r.provider.client.UpdateTopicConfig(data.Id.Value, topicConfigRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update topic config, got error: %s", err))
		return
	}

	mapTopicConfigResponseToData(ctx, &data, topicConfig)
	tflog.Trace(ctx, "Updated a topic config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r topicConfigResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data topicConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteTopicConfig(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete topic config, got error: %s", err))
		return
	}
}

func (r topicConfigResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createTopicConfigRequestFromData(ctx context.Context, data *topicConfigResourceData, r topicConfigResource) (webclient.TopicConfigRequest, error) {
	rawTopic, err := data.Topic.ToTerraformValue(ctx)
	if err != nil {
		return webclient.TopicConfigRequest{}, err
	}
	var topic string
	err = rawTopic.As(&topic)
	if err != nil {
		return webclient.TopicConfigRequest{}, err
	}
	topic = fmt.Sprintf("%s/streams/%v", r.provider.client.ApiURL, topic)

	rawEnvironment, err := data.Environment.ToTerraformValue(ctx)
	if err != nil {
		return webclient.TopicConfigRequest{}, err
	}
	var environment string
	err = rawEnvironment.As(&environment)
	if err != nil {
		return webclient.TopicConfigRequest{}, err
	}
	environment = fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, environment)

	topicConfigRequest := webclient.TopicConfigRequest{
		Partitions:    int(data.Partitions.Value),
		RetentionTime: int(data.RetentionTime.Value),
		Stream:        topic,
		Environment:   environment,
	}

	// optional fields
	if !data.KeySchemaVersion.Null {
		topicConfigRequest.KeySchemaVersion = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.KeySchemaVersion.Value)
	}
	if !data.ValueSchemaVersion.Null {
		topicConfigRequest.ValueSchemaVersion = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.ValueSchemaVersion.Value)
	}

	return topicConfigRequest, nil
}

func mapTopicConfigResponseToData(_ context.Context, data *topicConfigResourceData, topicConfig *webclient.TopicConfigResponse) {
	data.Id = types.String{Value: topicConfig.Uid}
	data.Partitions = types.Int64{Value: int64(topicConfig.Partitions)}
	data.RetentionTime = types.Int64{Value: int64(topicConfig.RetentionTime)}
	data.Topic = types.String{Value: topicConfig.Embedded.Stream.Uid}
	data.Environment = types.String{Value: topicConfig.Embedded.Environment.Uid}
	properties := make(map[string]attr.Value)
	for key, value := range topicConfig.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}
}

func (r topicConfigResource) validateSchemaVersionsForUpdate(schemaUid string, schemaVersionUid string, resp *tfsdk.UpdateResourceResponse) {
	keySchemaVersions, err := r.provider.client.GetSchemaVersionsBySchema(fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, schemaUid))
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource",
			fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	var isValidKeySchemaVersion = false
	for _, value := range keySchemaVersions.Embedded.SchemaVersion {
		if value.Uid == schemaVersionUid {
			isValidKeySchemaVersion = true
			break
		}
	}

	if !isValidKeySchemaVersion {
		resp.Diagnostics.AddError("CREATE request error for topic config resource",
			fmt.Sprintf("Error message: %s", schemaVersionUid+" is invalid schema id."))
		return
	}
}

func (r topicConfigResource) validateSchemaVersionsForCreate(schemaUid string, schemaVersionUid string, resp *tfsdk.CreateResourceResponse) {
	schemaVersions, err := r.provider.client.GetSchemaVersionsBySchema(fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, schemaUid))
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource",
			fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	var isValidKeySchemaVersion = false
	for _, value := range schemaVersions.Embedded.SchemaVersion {
		if value.Uid == schemaVersionUid {
			isValidKeySchemaVersion = true
			break
		}
	}

	if !isValidKeySchemaVersion {
		resp.Diagnostics.AddError("CREATE request error for topic config resource",
			fmt.Sprintf("Error message: %s", schemaVersionUid+" is invalid schema id."))
		return
	}
}

package provider

import (
	webclient "axual-webclient"
	"axual.com/terraform-provider-axual/internal/provider/utils"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"time"
)

var _ resource.Resource = &topicConfigResource{}
var _ resource.ResourceWithImportState = &topicConfigResource{}

func NewTopicConfigResource(provider AxualProvider) resource.Resource {
	return &topicConfigResource{
		provider: provider,
	}
}

type topicConfigResource struct {
	provider AxualProvider
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

func (r *topicConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic_config"
}

func (r *topicConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Topic Config resource. Once the Topic has been created, the next step to actually configure the topic for any environment is to configure the topic. Read more: https://docs.axual.io/axual/2025.1/self-service/topic-management.html#configuring-a-topic-for-an-environment",

		Attributes: map[string]schema.Attribute{
			"partitions": schema.Int64Attribute{
				MarkdownDescription: "The number of partitions define how many consumer instances can be started in parallel on this topic. Read more: https://docs.axual.io/axual/2025.1/self-service/topic-management.html#partitions-number",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"retention_time": schema.Int64Attribute{
				MarkdownDescription: "Determine how long the messages should be available on a topic. There should be an agreed value most likely discussed in Intake session with the team supporting Axual Platform. In most cases, it is 7 days. Minimum value is 1000 (ms). Read more: https://docs.axual.io/axual/2025.1/self-service/topic-management.html#retention-time",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1000),
				},
			},
			"topic": schema.StringAttribute{
				MarkdownDescription: "The Topic this topic configuration is associated with",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "The environment this topic configuration is associated with",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_schema_version": schema.StringAttribute{
				MarkdownDescription: "The schema version this topic config supports for the key.",
				Optional:            true,
			},
			"value_schema_version": schema.StringAttribute{
				MarkdownDescription: "The schema version this topic config supports for the value.",
				Optional:            true,
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "You can define Kafka properties for your topic here. All options are: `segment.ms`, `retention.bytes`, `min.compaction.lag.ms`, `max.compaction.lag.ms`, `message.timestamp.difference.max.ms`, `message.timestamp.type` Read more: https://docs.axual.io/axual/2025.1/self-service/topic-management.html#configuring-a-topic-for-an-environment",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("max.compaction.lag.ms", "message.timestamp.difference.max.ms", "message.timestamp.type", "min.compaction.lag.ms", "retention.bytes", "segment.ms")),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "topic config identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}
func (r *topicConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data topicConfigResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.GetTopic(data.Topic.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if !data.KeySchemaVersion.IsNull() {
		if topic.KeyType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Key Schema. Please don't set the KeySchemaVersion: %s", data.KeySchemaVersion.ValueString()))
			return
		} else {
			r.validateSchemaVersionsForCreate(topic.Embedded.KeySchema.Uid, data.KeySchemaVersion.ValueString(), resp)
			if resp.Diagnostics.HasError() {
				return
			}

		}
	}

	if !data.ValueSchemaVersion.IsNull() {
		if topic.ValueType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Value Schema. Please don't set the ValueSchemaVersion: %s", data.ValueSchemaVersion))
			return
		} else {
			r.validateSchemaVersionsForCreate(topic.Embedded.ValueSchema.Uid, data.ValueSchemaVersion.ValueString(), resp)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	topicConfigRequest, err := createTopicConfigRequestFromData(ctx, &data, r)
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elements() {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	topicConfigRequest.Properties = properties
	tflog.Info(ctx, fmt.Sprintf("Create topic config request %q", topicConfigRequest))

	var topicConfig *webclient.TopicConfigResponse
	// We retry to give time to Kafka to propagate changes
	retryErr := Retry(4, 5*time.Second, func() (err error) {
		topicConfig, err = r.provider.client.CreateTopicConfig(topicConfigRequest)
		return err
	})

	if retryErr != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message after retries: %s", retryErr.Error()))
		return
	}

	mapTopicConfigResponseToData(ctx, &data, topicConfig)
	tflog.Trace(ctx, "Created a topic config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data topicConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicConfig, err := r.provider.client.ReadTopicConfig(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Topic config not found. Id: %s", data.Id.ValueString()))
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

func (r *topicConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data topicConfigResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.GetTopic(data.Topic.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if !data.KeySchemaVersion.IsNull() {
		if topic.KeyType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Key Schema. Please don't set the KeySchemaVersion: %s", data.KeySchemaVersion.ValueString()))
			return
		} else {
			r.validateSchemaVersionsForUpdate(topic.Embedded.KeySchema.Uid, data.KeySchemaVersion.ValueString(), resp)
			if resp.Diagnostics.HasError() {
				return
			}

		}
	}

	if !data.ValueSchemaVersion.IsNull() {
		if topic.ValueType != "AVRO" {
			resp.Diagnostics.AddError(
				"CREATE request error for topic config resource",
				fmt.Sprintf("Topic doesn't have an AVRO Value Schema. Please don't set the ValueSchemaVersion: %s", data.ValueSchemaVersion))
			return
		} else {
			r.validateSchemaVersionsForUpdate(topic.Embedded.ValueSchema.Uid, data.ValueSchemaVersion.ValueString(), resp)
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
	req.State.GetAttribute(ctx, path.Root("properties"), &oldPropertiesState)

	properties := make(map[string]interface{})

	for key, _ := range oldPropertiesState {
		properties[key] = nil
	}

	for key, value := range data.Properties.Elements() {
		properties[key] = strings.Trim(value.String(), "\"")
	}

	topicConfigRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update topic config request %q", topicConfigRequest))

	// Retry logic for updating the topic config
	var topicConfig *webclient.TopicConfigResponse
	err = Retry(3, 2*time.Second, func() error {
		var updateErr error
		topicConfig, updateErr = r.provider.client.UpdateTopicConfig(data.Id.ValueString(), topicConfigRequest)
		return updateErr
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update topic config after retries, got error: %s", err))
		return
	}

	mapTopicConfigResponseToData(ctx, &data, topicConfig)
	tflog.Trace(ctx, "Updated a topic config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data topicConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Retry logic for deleting the topic config to give time for Kafka to propagate changes
	err := Retry(3, 3*time.Second, func() error {
		return r.provider.client.DeleteTopicConfig(data.Id.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete topic config after retries, got error: %s", err))
		return
	}
}

func (r *topicConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createTopicConfigRequestFromData(ctx context.Context, data *topicConfigResourceData, r *topicConfigResource) (webclient.TopicConfigRequest, error) {
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
		Partitions:    int(data.Partitions.ValueInt64()),
		RetentionTime: int(data.RetentionTime.ValueInt64()),
		Stream:        topic,
		Environment:   environment,
	}

	// optional fields
	if !data.KeySchemaVersion.IsNull() {
		topicConfigRequest.KeySchemaVersion = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.KeySchemaVersion.ValueString())
	}
	if !data.ValueSchemaVersion.IsNull() {
		topicConfigRequest.ValueSchemaVersion = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.ValueSchemaVersion.ValueString())
	}

	return topicConfigRequest, nil
}

func mapTopicConfigResponseToData(ctx context.Context, data *topicConfigResourceData, topicConfig *webclient.TopicConfigResponse) {
	data.Id = types.StringValue(topicConfig.Uid)
	data.Partitions = types.Int64Value(int64(topicConfig.Partitions))
	data.RetentionTime = types.Int64Value(int64(topicConfig.RetentionTime))
	data.Topic = types.StringValue(topicConfig.Embedded.Stream.Uid)
	data.Environment = types.StringValue(topicConfig.Embedded.Environment.Uid)
	data.Properties = utils.HandlePropertiesMapping(ctx, data.Properties, topicConfig.Properties)
}

func (r *topicConfigResource) validateSchemaVersionsForUpdate(schemaUid string, schemaVersionUid string, resp *resource.UpdateResponse) {
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

func (r *topicConfigResource) validateSchemaVersionsForCreate(schemaUid string, schemaVersionUid string, resp *resource.CreateResponse) {
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

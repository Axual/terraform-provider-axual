package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	custom_validator "axual.com/terraform-provider-axual/internal/custom-validator"
	"axual.com/terraform-provider-axual/internal/provider/utils"
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

var _ resource.Resource = &topicResource{}
var _ resource.ResourceWithImportState = &topicResource{}

func NewTopicResource(provider AxualProvider) resource.Resource {
	return &topicResource{
		provider: provider,
	}
}

type topicResource struct {
	provider AxualProvider
}

type topicResourceData struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	KeyType         types.String `tfsdk:"key_type"`
	KeySchema       types.String `tfsdk:"key_schema"`
	ValueType       types.String `tfsdk:"value_type"`
	ValueSchema     types.String `tfsdk:"value_schema"`
	Owners          types.String `tfsdk:"owners"`
	Viewers         types.Set    `tfsdk:"viewers"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Id              types.String `tfsdk:"id"`
	Properties      types.Map    `tfsdk:"properties"`
}

func (r *topicResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (r *topicResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A topic represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2024.4/self-service/topic-management.html",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the topic. Can only contain letters, numbers, dots, dashes and underscores and cannot begin with an underscore, dot or dash, but can't start with underscore, dot or dash. The topic name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 180),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`), "can only contain letters, numbers, dots, dashes and underscores, but cannot begin with an underscore, dot or dash"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A text describing the purpose of the topic.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"key_type": schema.StringAttribute{
				MarkdownDescription: "The key type and reference to the schema. Read more: https://docs.axual.io/axual/2024.4/self-service/topic-management.html#key-type",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("AVRO", "JSON", "Binary", "String", "Xml"),
				},
			},
			"key_schema": schema.StringAttribute{
				MarkdownDescription: "(if `key_type` is `AVRO`) The key type and reference to the schema (if applicable).",
				Optional:            true,
			},
			"value_type": schema.StringAttribute{
				MarkdownDescription: "The value type and reference to the schema. Read more: https://docs.axual.io/axual/2024.4/self-service/topic-management.html#value-type",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("AVRO", "JSON", "Binary", "String", "Xml"),
				},
			},
			"value_schema": schema.StringAttribute{
				MarkdownDescription: "(if `value_type` is `AVRO`) The value type and reference to the schema (if applicable).",
				Optional:            true,
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The team owning this topic. Read more: https://docs.axual.io/axual/2024.4/self-service/topic-management.html#topic-owner",
				Required:            true,
			},
			"viewers": schema.SetAttribute{
				MarkdownDescription: "The Viewer Groups of this topic. Topic Viewer Groups define which Groups are authorized to View Topic Configurations, regardless of ownership and visibility. Read more: https://docs.axual.io/axual/2024.4/self-service/user-group-management.html#viewer-groups",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					custom_validator.NewNonEmptySetValidator(),
				},
			},
			"retention_policy": schema.StringAttribute{
				MarkdownDescription: "Designate the retention policy to use on old log segments. Read more: https://docs.axual.io/axual/2024.4/self-service/topic-management.html#retention-policy",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("compact", "delete", "compact,delete", "delete,compact"),
				},
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "Advanced (Kafka) properties for a topic in a given environment. If no properties please leave properties empty like this: properties = { }.  Read more: https://docs.axual.io/axual/2024.4/self-service/advanced-features.html#configuring-topic-properties",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Topic unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *topicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data topicResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicRequest, err := createTopicRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	if data.Properties.Elements() != nil {
		properties := make(map[string]interface{})
		for key, value := range data.Properties.Elements() {
			properties[key] = strings.Trim(value.String(), "\"")
		}
		topicRequest.Properties = properties
	}

	tflog.Info(ctx, fmt.Sprintf("Create topic request %q", topicRequest))
	topic, err := r.provider.client.CreateTopic(topicRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapTopicResponseToData(ctx, &data, topic)
	tflog.Trace(ctx, "Created a topic resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.GetTopic(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Topic not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("READ request error for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapTopicResponseToData(ctx, &data, topic)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data topicResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	topicRequest, err := createTopicRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
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

	topicRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update topic request %q", topicRequest))
	topic, err := r.provider.client.UpdateTopic(data.Id.ValueString(), topicRequest)
	if err != nil {
		resp.Diagnostics.AddError("UPDATE request error for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapTopicResponseToData(ctx, &data, topic)
	tflog.Trace(ctx, "Updated a topic resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Retry logic for deleting the topic to give time for Kafka to propagate changes
	err := Retry(3, 3*time.Second, func() error {
		return r.provider.client.DeleteTopic(data.Id.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for topic resource", fmt.Sprintf("Error message after retries: %s", err.Error()))
		return
	}
}

func (r *topicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createTopicRequestFromData(ctx context.Context, data *topicResourceData, r *topicResource) (webclient.TopicRequest, error) {
	rawOwners, err := data.Owners.ToTerraformValue(ctx)
	if err != nil {
		return webclient.TopicRequest{}, err
	}
	var owners string
	err = rawOwners.As(&owners)
	if err != nil {
		return webclient.TopicRequest{}, err
	}
	owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)

	var keySchema string
	if data.KeyType.ValueString() == "AVRO" {
		if !data.KeySchema.IsNull() {
			keySchema = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.KeySchema.ValueString())
		} else {
			return webclient.TopicRequest{}, fmt.Errorf("KeyType is AVRO but KeySchema is null")
		}
	}

	var valueSchema string
	if data.ValueType.ValueString() == "AVRO" {
		if !data.ValueSchema.IsNull() {
			valueSchema = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.ValueSchema.ValueString())
		} else {
			return webclient.TopicRequest{}, fmt.Errorf("ValueType is AVRO but ValueSchema is null")
		}
	}

	viewers := []string{}
	if !data.Viewers.IsNull() {
		var viewerUIDs []string
		diags := data.Viewers.ElementsAs(ctx, &viewerUIDs, false)
		if diags.HasError() {
			return webclient.TopicRequest{}, fmt.Errorf("failed to extract viewers: %v", diags)
		}

		for _, viewer := range viewerUIDs {
			fullURL := fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, viewer)
			viewers = append(viewers, fullURL)
		}
	}

	topicRequest := webclient.TopicRequest{
		Name:            data.Name.ValueString(),
		KeyType:         data.KeyType.ValueString(),
		KeySchema:       keySchema,
		ValueType:       data.ValueType.ValueString(),
		ValueSchema:     valueSchema,
		Owners:          owners,
		Viewers:         viewers,
		RetentionPolicy: data.RetentionPolicy.ValueString(),
	}

	// optional fields
	if !data.Description.IsNull() {
		topicRequest.Description = data.Description.ValueString()
	}
	return topicRequest, nil
}

func mapTopicResponseToData(ctx context.Context, data *topicResourceData, topic *webclient.TopicResponse) {
	data.Id = types.StringValue(topic.Uid)
	data.Name = types.StringValue(topic.Name)
	data.KeyType = types.StringValue(topic.KeyType)
	data.ValueType = types.StringValue(topic.ValueType)
	data.Owners = types.StringValue(topic.Embedded.Owners.Uid)
	data.RetentionPolicy = types.StringValue(topic.RetentionPolicy)
	data.Properties = utils.HandlePropertiesMapping(ctx, data.Properties, topic.Properties)

	// Optional fields
	if topic.Description == nil || len(topic.Description.(string)) == 0 {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(topic.Description.(string))
	}

	if topic.Embedded.Viewers == nil || len(topic.Embedded.Viewers) == 0 {
		data.Viewers = types.SetNull(types.StringType)
	} else {
		viewerSet := make([]attr.Value, len(topic.Embedded.Viewers))
		for i, viewer := range topic.Embedded.Viewers {
			viewerSet[i] = types.StringValue(viewer.Uid)
		}
		viewers, diags := types.SetValue(types.StringType, viewerSet)
		if diags.HasError() {
			tflog.Error(ctx, "Error creating viewers set")
		}
		data.Viewers = viewers
	}

	if data.KeyType.ValueString() == "AVRO" {
		data.KeySchema = utils.SetStringValue(topic.Embedded.KeySchema.Uid)
	} else {
		data.KeySchema = types.StringNull()
	}

	if data.ValueType.ValueString() == "AVRO" {
		data.ValueSchema = utils.SetStringValue(topic.Embedded.ValueSchema.Uid)
	} else {
		data.ValueSchema = types.StringNull()
	}
}

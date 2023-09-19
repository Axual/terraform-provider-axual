package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = topicResourceType{}
var _ tfsdk.Resource = topicResource{}
var _ tfsdk.ResourceWithImportState = topicResource{}

type topicResourceType struct{}

func (t topicResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A topic represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html",

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
				Validators: []tfsdk.AttributeValidator{
					validation.Length(1, -1),
				},
			},
			"key_type": {
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#key-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"AVRO", "JSON", "Binary", "String", "Xml"}),
				},
			},
			"key_schema": {
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#key-schema",
				Optional:            true,
				Type:                types.StringType,
			},
			"value_type": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#value-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"AVRO", "JSON", "Binary", "String", "Xml"}),
				},
			},
			"value_schema": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#value-schema",
				Optional:            true,
				Type:                types.StringType,
			},
			"owners": {
				MarkdownDescription: "The team owning this topic. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#topic-owner",
				Required:            true,
				Type:                types.StringType,
			},
			"retention_policy": {
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#retention-policy",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"compact", "delete"}),
				},
			},
			"properties": {
				MarkdownDescription: "Advanced (Kafka) properties for a topic in a given environment. Read more: https://docs.axual.io/axual/2023.2/self-service/advanced-features.html#configuring-topic-properties",
				Required:            true,
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

func (t topicResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return topicResource{
		provider: provider,
	}, diags
}

type topicResourceData struct {
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

type topicResource struct {
	provider provider
}

func (r topicResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
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
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	topicRequest.Properties = properties

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

func (r topicResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.provider.client.ReadTopic(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Topic not found. Id: %s", data.Id.Value))
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

func (r topicResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
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
	req.State.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("properties"), &oldPropertiesState)

	properties := make(map[string]interface{})

	for key, _ := range oldPropertiesState {
		properties[key] = nil
	}

	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}

	topicRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update topic request %q", topicRequest))
	topic, err := r.provider.client.UpdateTopic(data.Id.Value, topicRequest)
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

func (r topicResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteTopic(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for topic resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func (r topicResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createTopicRequestFromData(ctx context.Context, data *topicResourceData, r topicResource) (webclient.TopicRequest, error) {
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
	if data.KeyType.Value == "AVRO" {
		if !data.KeySchema.Null {
			keySchema = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.KeySchema.Value)
		} else {
			return webclient.TopicRequest{}, fmt.Errorf("KeyType is AVRO but KeySchema is null")
		}
	}

	var valueSchema string
	if data.ValueType.Value == "AVRO" {
		if !data.ValueSchema.Null {
			valueSchema = fmt.Sprintf("%s/schemas/%v", r.provider.client.ApiURL, data.ValueSchema.Value)
		} else {
			return webclient.TopicRequest{}, fmt.Errorf("ValueType is AVRO but ValueSchema is null")
		}
	}

	topicRequest := webclient.TopicRequest{
		Name:            data.Name.Value,
		KeyType:         data.KeyType.Value,
		KeySchema:       keySchema,
		ValueType:       data.ValueType.Value,
		ValueSchema:     valueSchema,
		Owners:          owners,
		RetentionPolicy: data.RetentionPolicy.Value,
	}

	// optional fields
	if !data.Description.Null {
		topicRequest.Description = data.Description.Value
	}
	return topicRequest, nil
}

func mapTopicResponseToData(_ context.Context, data *topicResourceData, topic *webclient.TopicResponse) {
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

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
)

var _ tfsdk.ResourceType = streamResourceType{}
var _ tfsdk.Resource = streamResource{}
var _ tfsdk.ResourceWithImportState = streamResource{}

type streamResourceType struct{}

func (t streamResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A stream represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "The name of the stream. This must be in the format string-string (Needs to contain exactly one dash). The stream name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the stream.",
				Optional:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(1, -1),
				},
			},
			"key_type": {
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#key-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"JSON", "Binary", "String", "Xml"}),
				},
			},
			"value_type": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#value-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"JSON", "Binary", "String", "Xml"}),
				},
			},
			"owners": {
				MarkdownDescription: "The team owning this stream. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#stream-owner",
				Required:            true,
				Type:                types.StringType,
			},
			"retention_policy": {
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#retention-policy",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"compact", "delete"}),
				},
			},
			"properties": {
				MarkdownDescription: "Advanced (Kafka) properties for a stream in a given environment. Read more: https://docs.axual.io/axual/2023.2/self-service/advanced-features.html#configuring-stream-properties",
				Required:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Stream unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t streamResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return streamResource{
		provider: provider,
	}, diags
}

type streamResourceData struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	KeyType         types.String `tfsdk:"key_type"`
	ValueType       types.String `tfsdk:"value_type"`
	Owners          types.String `tfsdk:"owners"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Id              types.String `tfsdk:"id"`
	Properties      types.Map    `tfsdk:"properties"`
}

type streamResource struct {
	provider provider
}

func (r streamResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data streamResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	streamRequest, err := createStreamRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	streamRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Create stream request %q", streamRequest))
	stream, err := r.provider.client.CreateStream(streamRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapStreamResponseToData(ctx, &data, stream)
	tflog.Trace(ctx, "Created a stream resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data streamResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.provider.client.ReadStream(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Stream not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("READ request error for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapStreamResponseToData(ctx, &data, stream)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data streamResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	streamRequest, err := createStreamRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
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

	streamRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update stream request %q", streamRequest))
	stream, err := r.provider.client.UpdateStream(data.Id.Value, streamRequest)
	if err != nil {
		resp.Diagnostics.AddError("UPDATE request error for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapStreamResponseToData(ctx, &data, stream)
	tflog.Trace(ctx, "Updated a stream resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data streamResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteStream(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for stream resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func (r streamResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createStreamRequestFromData(ctx context.Context, data *streamResourceData, r streamResource) (webclient.StreamRequest, error) {
	rawOwners, err := data.Owners.ToTerraformValue(ctx)
	if err != nil {
		return webclient.StreamRequest{}, err
	}
	var owners string
	err = rawOwners.As(&owners)
	if err != nil {
		return webclient.StreamRequest{}, err
	}
	owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)

	streamRequest := webclient.StreamRequest{
		Name:            data.Name.Value,
		KeyType:         data.KeyType.Value,
		ValueType:       data.ValueType.Value,
		Owners:          owners,
		RetentionPolicy: data.RetentionPolicy.Value,
	}

	// optional fields
	if !data.Description.Null {
		streamRequest.Description = data.Description.Value
	}
	return streamRequest, nil
}

func mapStreamResponseToData(_ context.Context, data *streamResourceData, stream *webclient.StreamResponse) {
	data.Id = types.String{Value: stream.Uid}
	data.Name = types.String{Value: stream.Name}
	data.KeyType = types.String{Value: stream.KeyType}
	data.ValueType = types.String{Value: stream.ValueType}
	data.Owners = types.String{Value: stream.Embedded.Owners.Uid}
	data.RetentionPolicy = types.String{Value: stream.RetentionPolicy}

	properties := make(map[string]attr.Value)
	for key, value := range stream.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}

	// optional fields
	if stream.Description == nil || len(stream.Description.(string)) == 0 {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: stream.Description.(string)}
	}
}

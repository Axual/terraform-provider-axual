package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
)

var _ tfsdk.ResourceType = streamConfigResourceType{}
var _ tfsdk.Resource = streamConfigResource{}
var _ tfsdk.ResourceWithImportState = streamConfigResource{}

type streamConfigResourceType struct{}

func (t streamConfigResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Stream Config resource. Once the Stream has been created, the next step to actually configure the stream for any environment is to configure the stream. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#configuring-a-stream-for-an-environment",

		Attributes: map[string]tfsdk.Attribute{
			"partitions": {
				MarkdownDescription: "The number of partitions define how many consumer instances can be started in parallel on this stream. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#partitions-number",
				Required:            true,
				Type:                types.Int64Type,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"retention_time": {
				MarkdownDescription: "Determine how long the messages should be available on a stream. There should be an agreed value most likely discussed in Intake session with the team supporting Axual Platform. In most cases, it is 7 days. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#retention-time",
				Required:            true,
				Type:                types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					int64validator.AtLeast(1000),
				},
			},
			"stream": {
				MarkdownDescription: "The Stream this stream configuration is associated with",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"environment": {
				MarkdownDescription: "The environment this stream configuration is associated with",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"properties": {
				MarkdownDescription: "You can define Kafka properties for your stream here. segment.ms property needs to always be included. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#configuring-a-stream-for-an-environment",
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
				MarkdownDescription: "stream config identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t streamConfigResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return streamConfigResource{
		provider: provider,
	}, diags
}

type streamConfigResourceData struct {
	Partitions    types.Int64  `tfsdk:"partitions"`
	RetentionTime types.Int64  `tfsdk:"retention_time"`
	Stream        types.String `tfsdk:"stream"`
	Environment   types.String `tfsdk:"environment"`
	Id            types.String `tfsdk:"id"`
	Properties    types.Map    `tfsdk:"properties"`
}

type streamConfigResource struct {
	provider provider
}

func (r streamConfigResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data streamConfigResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	streamConfigRequest, err := createStreamConfigRequestFromData(ctx, &data, r)
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	streamConfigRequest.Properties = properties
	tflog.Info(ctx, fmt.Sprintf("Create stream config request %q", streamConfigRequest))
	streamConfig, err := r.provider.client.CreateStreamConfig(streamConfigRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for stream config resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapStreamConfigResponseToData(ctx, &data, streamConfig)
	tflog.Trace(ctx, "Created a stream config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamConfigResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data streamConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	streamConfig, err := r.provider.client.ReadStreamConfig(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Stream config not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read stream config, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapStreamConfigResponseToData(ctx, &data, streamConfig)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamConfigResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data streamConfigResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	streamConfigRequest, err := createStreamConfigRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for stream config resource", fmt.Sprintf("Error message: %s", err.Error()))
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

	streamConfigRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update stream config request %q", streamConfigRequest))
	streamConfig, err := r.provider.client.UpdateStreamConfig(data.Id.Value, streamConfigRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update stream config, got error: %s", err))
		return
	}

	mapStreamConfigResponseToData(ctx, &data, streamConfig)
	tflog.Trace(ctx, "Updated a stream config resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r streamConfigResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data streamConfigResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteStreamConfig(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete stream config, got error: %s", err))
		return
	}
}

func (r streamConfigResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createStreamConfigRequestFromData(ctx context.Context, data *streamConfigResourceData, r streamConfigResource) (webclient.StreamConfigRequest, error) {
	rawStream, err := data.Stream.ToTerraformValue(ctx)
	if err != nil {
		return webclient.StreamConfigRequest{}, err
	}
	var stream string
	err = rawStream.As(&stream)
	if err != nil {
		return webclient.StreamConfigRequest{}, err
	}
	stream = fmt.Sprintf("%s/streams/%v", r.provider.client.ApiURL, stream)

	rawEnvironment, err := data.Environment.ToTerraformValue(ctx)
	if err != nil {
		return webclient.StreamConfigRequest{}, err
	}
	var environment string
	err = rawEnvironment.As(&environment)
	if err != nil {
		return webclient.StreamConfigRequest{}, err
	}
	environment = fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, environment)

	streamConfigRequest := webclient.StreamConfigRequest{
		Partitions:    int(data.Partitions.Value),
		RetentionTime: int(data.RetentionTime.Value),
		Stream:        stream,
		Environment:   environment,
	}
	return streamConfigRequest, nil
}

func mapStreamConfigResponseToData(_ context.Context, data *streamConfigResourceData, streamConfig *webclient.StreamConfigResponse) {
	data.Id = types.String{Value: streamConfig.Uid}
	data.Partitions = types.Int64{Value: int64(streamConfig.Partitions)}
	data.RetentionTime = types.Int64{Value: int64(streamConfig.RetentionTime)}
	data.Stream = types.String{Value: streamConfig.Embedded.Stream.Uid}
	data.Environment = types.String{Value: streamConfig.Embedded.Environment.Uid}
	properties := make(map[string]attr.Value)
	for key, value := range streamConfig.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}
}

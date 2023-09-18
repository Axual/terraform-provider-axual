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
var _ tfsdk.DataSourceType = streamDataSourceType{}
var _ tfsdk.DataSource = streamDataSource{}

type streamDataSourceType struct{}

func (t streamDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A stream represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "The name of the stream. This must be in the format string-string (Needs to contain exactly one dash). The stream name is usually discussed and finalized as part of the Intake session or a follow up.",
				Computed:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the stream.",
				Optional:            true,
				Type:                types.StringType,
				Computed:            true,
			},
			"key_type": {
				MarkdownDescription: "The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#key-type",
				Type:                types.StringType,
				Computed:            true,
			},
			"value_type": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#value-type",
				
				Type:                types.StringType,
				Computed:            true,
			},
			"owners": {
				MarkdownDescription: "The team owning this stream. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#stream-owner",
				Type:                types.StringType,
				Computed:            true,
			},
			"retention_policy": {
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2023.2/self-service/stream-management.html#retention-policy",
				Type:                types.StringType,
				Computed:            true,
			},
			"properties": {
				MarkdownDescription: "Advanced (Kafka) properties for a stream in a given environment. Read more: https://docs.axual.io/axual/2023.2/self-service/advanced-features.html#configuring-stream-properties",
				Computed:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"id": {
				Required:            true,
				MarkdownDescription: "Stream unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t streamDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return streamDataSource{
		provider: provider,
	}, diags
}

type streamDataSourceData struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	KeyType         types.String `tfsdk:"key_type"`
	ValueType       types.String `tfsdk:"value_type"`
	Owners          types.String `tfsdk:"owners"`
	RetentionPolicy types.String `tfsdk:"retention_policy"`
	Id              types.String `tfsdk:"id"`
	Properties      types.Map    `tfsdk:"properties"`
}

type streamDataSource struct {
	provider provider
}

func (d streamDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data streamDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := d.provider.client.ReadStream(data.Id.Value)
	if err != nil {
	    resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read stream, got error: %s", err))
	return
	}

    mapStreamDataSourceResponseToData(ctx, &data, stream)
	
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapStreamDataSourceResponseToData(ctx context.Context, data *streamDataSourceData, stream *webclient.StreamResponse) {

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

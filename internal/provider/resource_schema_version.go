package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var(
	_ tfsdk.ResourceType = schemaVersionResourceType{}
	_ tfsdk.Resource = schemaVersionResource{}
 	_ tfsdk.ResourceWithImportState = schemaVersionResource{}
)

type schemaVersionResourceType struct{}

func (r schemaVersionResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Schema version resource. Read more: https://docs.axual.io/axual/2023.2/self-service/schema-management.html",

		Attributes: map[string]tfsdk.Attribute{
			"schema": {
				MarkdownDescription: "Avro schema",
				Required:            true,
				Type:                types.StringType,
				// ! TODO Add validation for avro schema
	
			},
			"version": {
				MarkdownDescription: "The version of the schema",
				Required:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A short text describing the Schema version",
				Optional:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(0, 500),
				},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Schema version unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.RegexpMatch(`^[0-9a-fA-F]{32}$`),
				},
			},
		},
	}, nil
}

func (r schemaVersionResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return schemaVersionResource{
		provider: provider,
	}, diags
}

type schemaVersionResourceData struct {
	Schema    types.String `tfsdk:"schema"`
	Version    types.String `tfsdk:"version"`
	Description   types.String `tfsdk:"description"`
	Id           types.String `tfsdk:"id"`
}


type schemaVersionResource struct {
	provider provider
}

func (r schemaVersionResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data schemaResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	schemaRequest := createSchemaRequestFromData(ctx, &data)

	schema, err := r.provider.client.CreateSchema(schemaRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for schema resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapSchemaResponseToData(ctx, &data, schema)
	tflog.Trace(ctx, "created a resource")
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r schemaVersionResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data schemaResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	schema, err := r.provider.client.GetSchema(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Schema not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapSchemaResponseToData(ctx, &data, schema)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r schemaVersionResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data schemaVersionResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	schemaRequest := createSchemaRequestFromData(ctx, &data)

	schema, err := r.provider.client.UpdateSchema(data.Id.Value, schemaRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update schema, got error: %s", err))
		return
	}

	mapSchemaResponseToData(ctx, &data, schema)
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r schemaVersionResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	resp.Diagnostics.AddError("Client Error", "Deleting of schema is not allowed!")

}

func (r schemaVersionResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func mapSchemaResponseToData(_ context.Context, data *schemaVersionResourceData, schema *webclient.SchemaVersionCreateResponse) {
	// mandatory fields first
	data.Id = types.String{Value: schema.Uid}
	data.Name = types.String{Value: schema.Name}
	
	// optional fields
	if schema.Description == nil {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: schema.Description.(string)}
	}

	
}

func createSchemaRequestFromData(ctx context.Context, data *schemaVersionResourceData) webclient.SchemaVersionRequest {

	r := webclient.SchemaVersionRequest{
	Schema: data.Schema.Value,
	Version: data.Version.Value,
	}

	// optional fields
	if(!data.Description.Null) {
		r.Description = data.Description.Value
	}

	tflog.Info(ctx, fmt.Sprintf("schema request %q", r))
	return r
}

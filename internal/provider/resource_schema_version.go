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

			"schema_uid": {
				Computed:            true,
				MarkdownDescription: "Schema unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.RegexpMatch(`^[0-9a-fA-F]{32}$`),
				},
			},

			"full_name": {
				Computed:            true,
				MarkdownDescription: "Full name of the schema",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
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
	Id types.String`tfsdk:"id"`
	SchemaUid types.String`tfsdk:"schema_uid"`
	FullName types.String`tfsdk:"full_name"`
}

type schemaVersionResource struct {
	provider provider
}

func (r schemaVersionResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data schemaVersionResourceData
	
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	
	vsReq:= createValidateSchemaVersionRequestFromData(ctx, &data)

	fmt.Printf("vsReq.Schema: %v\n", vsReq.Schema)
	valid, valErr:= r.provider.client.ValidateSchemaVersion(vsReq)

	if(valErr!=nil) {
		resp.Diagnostics.AddError("Validate Schema request error for schema version resource", fmt.Sprintf("Error message: %s", valErr.Error()))
		return	
	}

	svReq := createSchemaVersionRequestFromData(ctx, valid , &data,)

	svResp, err := r.provider.client.CreateSchemaVersion(svReq)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for schema version resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapCreateSchemaVersionResponseToData(ctx, &data, svResp)
	tflog.Trace(ctx, "created a resource")
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r schemaVersionResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data schemaVersionResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	svResp, err := r.provider.client.GetSchemaVersion(data.Id.Value)
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
	mapGetSchemaVersionResponseToData(ctx, &data, svResp)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r schemaVersionResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	
	resp.Diagnostics.AddError("Client Error", "API does not allow update of schema version")
}

func (r schemaVersionResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	resp.Diagnostics.AddError("Client Error", "Deleting of schema is not allowed!")

}

func (r schemaVersionResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func mapCreateSchemaVersionResponseToData(_ context.Context, data *schemaVersionResourceData, resp *webclient.CreateSchemaVersionResponse) {

	data.SchemaUid = types.String{Value: resp.SchemaUid}
	data.Id = types.String{Value: resp.Id}
	data.FullName = types.String{Value: resp.FullName}
	data.Version = types.String{Value: resp.Version}	
}
func mapGetSchemaVersionResponseToData(_ context.Context, data *schemaVersionResourceData, resp *webclient.GetSchemaVersionResponse) {

	data.SchemaUid = types.String{Value: resp.Schema.SchemaUid}
	data.Id = types.String{Value: resp.Id}
	data.FullName = types.String{Value: resp.Schema.Name}
	data.Version = types.String{Value: resp.Version}	
}

func createValidateSchemaVersionRequestFromData(ctx context.Context, data *schemaVersionResourceData) webclient.ValidateSchemaVersionRequest {

	r := webclient.ValidateSchemaVersionRequest{
	Schema: data.Schema.Value,
	}

	tflog.Info(ctx, fmt.Sprintf("schema version request %q", r))
	return r
}

func createSchemaVersionRequestFromData(ctx context.Context, parsedSchema *webclient.ValidateSchemaVersionResponse, data *schemaVersionResourceData) webclient.SchemaVersionRequest {

	r := webclient.SchemaVersionRequest{
	Schema: parsedSchema.Schema,
	Version: data.Version.Value,
	}

	// optional fields
	if(!data.Description.Null) {
		r.Description = data.Description.Value
	}

	tflog.Info(ctx, fmt.Sprintf("schema version request %q", r))
	return r
}

package provider

import (
	"context"
	"fmt"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = schemaVersionDataSourceType{}
var _ tfsdk.DataSource = schemaVersionDataSource{}

type schemaVersionDataSourceType struct{}

func (t schemaVersionDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Schema version resource. Read more: https://docs.axual.io/axual/2024.1/self-service/schema-management.html",

		Attributes: map[string]tfsdk.Attribute{
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
			"body": {
				MarkdownDescription: "Avro schema",
				Computed:            true,
				Type:                types.StringType,
			},
			"version": {
				MarkdownDescription: "The version of the schema",
				Required:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A short text describing the Schema version",
				Computed:            true,
				Type:                types.StringType,
			},
			"schema_id": {
				Computed:            true,
				MarkdownDescription: "Schema unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},

			"full_name": {
				Required:            true,
				MarkdownDescription: "Full name of the schema",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t schemaVersionDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return schemaVersionDataSource{
		provider: provider,
	}, diags
}

type schemaVersionDataSourceData struct {
	Body        types.String `tfsdk:"body"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	SchemaId    types.String `tfsdk:"schema_id"`
	FullName    types.String `tfsdk:"full_name"`
}

type schemaVersionDataSource struct {
	provider provider
}

func (d schemaVersionDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data schemaVersionDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	schema, err := d.provider.client.GetSchemaByName(data.FullName.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema version, got error: %s", err))
		return
	}

	sv, err2 := d.provider.client.GetSchemaVersionsBySchema(schema.Embedded.Schemas[0].Links.Self.Href)

	if err2 != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema version, got error: %s", err2))
		return
	}

	foundMatchingVersion := false

	for i := range sv.Embedded.SchemaVersion {
		if sv.Embedded.SchemaVersion[i].Version == data.Version.Value {
			foundMatchingVersion = true
			data.Id = types.String{Value: sv.Embedded.SchemaVersion[i].Uid}
			data.Version = types.String{Value: sv.Embedded.SchemaVersion[i].Version}
			data.Body = types.String{Value: sv.Embedded.SchemaVersion[i].SchemaBody}
			data.SchemaId = types.String{Value: sv.Embedded.SchemaVersion[i].Embedded.Schema.Uid}
			data.FullName = types.String{Value: sv.Embedded.SchemaVersion[i].Embedded.Schema.Name}
			data.Description = types.String{Value: sv.Embedded.SchemaVersion[i].Embedded.Schema.Description}
		}
	}

	if !foundMatchingVersion {
		resp.Diagnostics.AddError("Client Error", "Schema version matching the name you requested was not found")
	}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

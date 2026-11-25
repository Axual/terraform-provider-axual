package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &schemaVersionDataSource{}
var _ datasource.DataSourceWithValidateConfig = &schemaVersionDataSource{}

func NewSchemaVersionDataSource(provider AxualProvider) datasource.DataSource {
	return &schemaVersionDataSource{
		provider: provider,
	}
}

type schemaVersionDataSource struct {
	provider AxualProvider
}

type schemaVersionDataSourceData struct {
	Body        types.String `tfsdk:"body"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	SchemaId    types.String `tfsdk:"schema_id"`
	FullName    types.String `tfsdk:"full_name"`
	Owners      types.String `tfsdk:"owners"`
}

func (d *schemaVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_version"
}

func (d *schemaVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schema version resource. Read more: https://docs.axual.io/axual/2026.1/self-service/schema-management.html",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schema version unique identifier",
			},
			"body": schema.StringAttribute{
				MarkdownDescription: "The schema body: Avro, Protobuf or JSON Schema.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The schema version to read, for example `1.0.0`.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the schema: the latest one set on any version, not always the one of `version`.",
				Computed:            true,
			},
			"schema_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schema unique identifier",
			},
			"full_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Full name of the schema: `<namespace>.<name>`, for example `io.axual.qa.general.GitOpsTest`.",
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The uid of the Group that owns the schema. Read-only: a value set here is ignored.",
				Optional:            true,
				Computed:            false,
			},
		},
	}
}

func (d *schemaVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data schemaVersionDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	axualSchema, err := d.provider.client.GetSchemaByName(data.FullName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read axualSchema version, got error: %s", err))
		return
	}
	if len(axualSchema.Embedded.Schemas) == 0 {
		addNotFoundError(&resp.Diagnostics, "Schema", "full_name", data.FullName.ValueString())
		return
	}

	sv, err2 := d.provider.client.GetSchemaVersionsBySchema(axualSchema.Embedded.Schemas[0].Links.Self.Href)

	if err2 != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read axualSchema version, got error: %s", err2))
		return
	}

	// data.Version and data.FullName are overwritten below, so capture them before the loop starts.
	requestedVersion := data.Version.ValueString()
	requestedFullName := data.FullName.ValueString()
	foundMatchingVersion := false

	for i := range sv.Embedded.SchemaVersion {
		if sv.Embedded.SchemaVersion[i].Version != requestedVersion {
			continue
		}
		foundMatchingVersion = true
		data.Id = types.StringValue(sv.Embedded.SchemaVersion[i].Uid)
		data.Version = types.StringValue(sv.Embedded.SchemaVersion[i].Version)
		data.Body = types.StringValue(sv.Embedded.SchemaVersion[i].SchemaBody)
		data.SchemaId = types.StringValue(sv.Embedded.SchemaVersion[i].Embedded.Schema.Uid)
		data.FullName = types.StringValue(sv.Embedded.SchemaVersion[i].Embedded.Schema.Name)
		data.Description = types.StringValue(sv.Embedded.SchemaVersion[i].Embedded.Schema.Description)
		if sv.Embedded.SchemaVersion[i].Embedded.Schema.Owners != nil {
			data.Owners = types.StringValue(sv.Embedded.SchemaVersion[i].Embedded.Schema.Owners.UID)
		}
		break
	}

	if !foundMatchingVersion {
		resp.Diagnostics.AddError("Resource Not Found", fmt.Sprintf("No version '%s' found for schema with full_name '%s'.", requestedVersion, requestedFullName))
		return
	}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// ValidateConfig warns when owners is set. The data source does not filter by owner: it reads the owner
// of the schema, so a value set here is ignored. A DeprecationMessage would also warn on every read of
// owners, which is fine to do.
func (d *schemaVersionDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var owners types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("owners"), &owners)...)
	if !owners.IsNull() {
		resp.Diagnostics.AddAttributeWarning(path.Root("owners"), "owners is ignored",
			"The axual_schema_version data source does not filter by owner: it reads the owner of the schema. Remove owners from the data source block.")
	}
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &schemaVersionResource{}

func NewSchemaVersionResource(provider AxualProvider) resource.Resource {
	return &schemaVersionResource{
		provider: provider,
	}
}

type schemaVersionResource struct {
	provider AxualProvider
}

type schemaVersionResourceData struct {
	Body        types.String `tfsdk:"body"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	SchemaId    types.String `tfsdk:"schema_id"`
	FullName    types.String `tfsdk:"full_name"`
	Owners      types.String `tfsdk:"owners"`
}

func (r *schemaVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_version"
}

func (r *schemaVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schema version resource. None of the fields can be update. Read more: https://docs.axual.io/axual/2024.2/self-service/schema-management.html",

		Attributes: map[string]schema.Attribute{
			"body": schema.StringAttribute{
				MarkdownDescription: "Avro schema",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the schema",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short text describing the Schema version",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schema version unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schema unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"full_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of the schema",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "Schema Owner",
				Optional:            true,
				Computed:            false,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},
		},
	}
}

func (r *schemaVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data schemaVersionResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	vsReq := createValidateSchemaVersionRequestFromData(ctx, &data)
	valid, valErr := r.provider.client.ValidateSchemaVersion(vsReq)

	const errorMsg = "Error message: %s"

	if valErr != nil {
		resp.Diagnostics.AddError("Validate Schema request error for schema version resource", fmt.Sprintf(errorMsg, valErr.Error()))
		return
	}

	svReq, err := createSchemaVersionRequestFromData(ctx, valid, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for schemaVersion resource", fmt.Sprintf(errorMsg, err.Error()))
		return
	}
	svResp, err := r.provider.client.CreateSchemaVersion(svReq)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for schema version resource", fmt.Sprintf(errorMsg, err.Error()))
		return
	}

	mapCreateSchemaVersionResponseToData(ctx, &data, svResp)
	tflog.Trace(ctx, "created a resource")
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *schemaVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data schemaVersionResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	svResp, err := r.provider.client.GetSchemaVersion(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Schema version not found. Id: %s", data.Id.ValueString()))
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

func (r *schemaVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Client Error", "API does not allow update of schema version. Please create another version of the schema")
}

func (r *schemaVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data schemaVersionResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteSchemaVersion(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for schema version resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func createValidateSchemaVersionRequestFromData(ctx context.Context, data *schemaVersionResourceData) webclient.ValidateSchemaVersionRequest {

	r := webclient.ValidateSchemaVersionRequest{
		Schema: data.Body.ValueString(),
	}

	tflog.Info(ctx, fmt.Sprintf("validating schema version request %q", r))
	return r
}

func createSchemaVersionRequestFromData(ctx context.Context, parsedSchema *webclient.ValidateSchemaVersionResponse, data *schemaVersionResourceData, r *schemaVersionResource) (webclient.SchemaVersionRequest, error) {

	schemaVersionRequest := webclient.SchemaVersionRequest{
		Schema:  parsedSchema.Schema,
		Version: data.Version.ValueString(),
	}

	if data.Owners.IsNull() || data.Owners.ValueString() == "" {
		var ownersPtr *string = nil
		schemaVersionRequest.Owners = ownersPtr
	} else {

		rawOwners, err := data.Owners.ToTerraformValue(ctx)
		if err != nil {
			return webclient.SchemaVersionRequest{}, err
		}
		var owners string
		err = rawOwners.As(&owners)
		if err != nil {
			return webclient.SchemaVersionRequest{}, err
		}
		owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)
		schemaVersionRequest.Owners = &owners
	}

	// optional fields
	if !data.Description.IsNull() {
		schemaVersionRequest.Description = data.Description.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("schema version request %q", schemaVersionRequest))
	return schemaVersionRequest, nil
}

func mapCreateSchemaVersionResponseToData(_ context.Context, data *schemaVersionResourceData, resp *webclient.CreateSchemaVersionResponse) {
	data.SchemaId = types.StringValue(resp.SchemaId)
	data.Id = types.StringValue(resp.Id)
	data.FullName = types.StringValue(resp.FullName)
	data.Version = types.StringValue(resp.Version)
	if resp.Owners == nil {
		data.Owners = types.StringNull()
	}
	if resp.Owners != nil && resp.Owners.ID != "" {
		data.Owners = types.StringValue(resp.Owners.ID)
	} else {
		data.Owners = types.StringNull()
	}
}
func mapGetSchemaVersionResponseToData(_ context.Context, data *schemaVersionResourceData, resp *webclient.GetSchemaVersionResponse) {
	data.SchemaId = types.StringValue(resp.Schema.SchemaId)
	data.Id = types.StringValue(resp.Id)
	data.FullName = types.StringValue(resp.Schema.Name)
	data.Version = types.StringValue(resp.Version)
	if resp.Schema.Owners == nil {
		data.Owners = types.StringNull()
	}
	if resp.Schema.Owners != nil && resp.Schema.Owners.ID != "" {
		data.Owners = types.StringValue(resp.Schema.Owners.ID)
	} else {
		data.Owners = types.StringNull()
	}
}

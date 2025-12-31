package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &schemaVersionResource{}
var _ resource.ResourceWithImportState = &topicResource{}

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
	Type        types.String `tfsdk:"type"`
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
		MarkdownDescription: "Schema version resource. None of the fields can be updated. Read more: https://docs.axual.io/axual/2025.3/self-service/schema-management.html",

		Attributes: map[string]schema.Attribute{
			"body": schema.StringAttribute{
				MarkdownDescription: "Schema definition. For AVRO schemas, provide valid JSON. For PROTOBUF, provide .proto file content. For JSON_SCHEMA, provide valid JSON Schema definition.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the schema",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short text describing the Schema",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the schema. Valid values are: AVRO, PROTOBUF, JSON_SCHEMA. Defaults to AVRO if not specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("AVRO", "PROTOBUF", "JSON_SCHEMA"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				MarkdownDescription: "The UID of the team owning this Schema",
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
			tflog.Warn(ctx, fmt.Sprintf("Schema version not found. Version ID: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema version, got error: %s", err))
		}
		return
	}
	tflog.Info(ctx, "Mapping the API response to the resource data")
	newData := schemaVersionResourceData{}
	mapGetSchemaVersionResponseToData(ctx, &data, &newData, svResp, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Saving the resource to state")

	diags = resp.State.Set(ctx, &newData)
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

func (r *schemaVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createValidateSchemaVersionRequestFromData(ctx context.Context, data *schemaVersionResourceData) webclient.ValidateSchemaVersionRequest {

	r := webclient.ValidateSchemaVersionRequest{
		Schema: data.Body.ValueString(),
	}

	// Add type field if specified
	if !data.Type.IsNull() && data.Type.ValueString() != "" {
		schemaType := data.Type.ValueString()
		r.Type = &schemaType
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

	// Add type field if specified
	if !data.Type.IsNull() && data.Type.ValueString() != "" {
		schemaType := data.Type.ValueString()
		schemaVersionRequest.Type = &schemaType
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

func mapGetSchemaVersionResponseToData(
	ctx context.Context,
	existingState *schemaVersionResourceData,
	newData *schemaVersionResourceData,
	resp *webclient.GetSchemaVersionResponse,
	diagnostics *diag.Diagnostics,
) {
	tflog.Info(ctx, "Mapping mandatory fields from API response to resource data.")
	newData.SchemaId = types.StringValue(resp.Schema.SchemaId)
	newData.Id = types.StringValue(resp.Id)
	newData.FullName = types.StringValue(resp.Schema.Name)
	newData.Version = types.StringValue(resp.Version)
	newData.Type = types.StringValue(resp.Schema.Type)

	tflog.Info(ctx, "Mapping optional fields.")
	if resp.Schema.Owners == nil || resp.Schema.Owners.ID == "" {
		tflog.Info(ctx, "Schema owners not found, setting to null.")
		newData.Owners = types.StringNull()
	} else {
		newData.Owners = types.StringValue(resp.Schema.Owners.ID)
	}

	tflog.Info(ctx, "Processing the schema body.")
	mapSchemaBody(ctx, existingState, newData, resp.SchemaBody, diagnostics)

	if resp.Schema.Description == "" {
		tflog.Info(ctx, "Schema description is empty, setting to null.")
		newData.Description = types.StringNull()
	} else {
		newData.Description = types.StringValue(resp.Schema.Description)
	}
}

func mapSchemaBody(
	ctx context.Context,
	existingState *schemaVersionResourceData,
	newData *schemaVersionResourceData,
	schemaBody string,
	diagnostics *diag.Diagnostics,
) {
	if schemaBody == "" {
		tflog.Info(ctx, "Schema body is empty, setting to null.")
		newData.Body = types.StringNull()
		return
	}

	// For JSON-based schemas (AVRO, JSON_SCHEMA), normalize JSON to avoid whitespace drift
	schemaType := newData.Type.ValueString()
	if schemaType == "AVRO" || schemaType == "JSON_SCHEMA" {
		tflog.Info(ctx, fmt.Sprintf("Normalizing JSON for schema type: %s", schemaType))

		// Check if existing state has a body value
		if !existingState.Body.IsNull() && !existingState.Body.IsUnknown() {
			// Try to normalize both existing and new bodies for comparison
			existingNormalized, existingErr := normalizeJSON(existingState.Body.ValueString())
			newNormalized, newErr := normalizeJSON(schemaBody)

			if existingErr == nil && newErr == nil && existingNormalized == newNormalized {
				tflog.Info(ctx, "Schema bodies are semantically equivalent (after JSON normalization). Preserving existing state.")
				newData.Body = existingState.Body
				return
			}
		}
	}

	tflog.Info(ctx, "Setting schema body from API response.")
	newData.Body = types.StringValue(schemaBody)
}

// normalizeJSON normalizes JSON by unmarshaling and marshaling with consistent formatting
func normalizeJSON(jsonStr string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

package provider

import (
	webclient "axual-webclient"
	"axual.com/terraform-provider-axual/internal/provider/utils"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	Body        jsontypes.Normalized `tfsdk:"body"`
	Version     types.String         `tfsdk:"version"`
	Description types.String         `tfsdk:"description"`
	Id          types.String         `tfsdk:"id"`
	SchemaId    types.String         `tfsdk:"schema_id"`
	FullName    types.String         `tfsdk:"full_name"`
	Owners      types.String         `tfsdk:"owners"`
}

func (r *schemaVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_version"
}

func (r *schemaVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schema version resource. None of the fields can be updated. Read more: https://docs.axual.io/axual/2024.2/self-service/schema-management.html",

		Attributes: map[string]schema.Attribute{
			"body": schema.StringAttribute{
				MarkdownDescription: "Avro schema as valid JSON",
				Required:            true,
				CustomType:          jsontypes.NormalizedType{},
				PlanModifiers: []planmodifier.String{
					utils.NormalizePlanModifier{},
				},
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
	newData.Description = types.StringValue(resp.Schema.Description)

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
		newData.Body = jsontypes.NewNormalizedNull()
		return
	}

	newBody := jsontypes.NewNormalizedValue(schemaBody)

	if !existingState.Body.IsNull() {
		tflog.Info(ctx, "Comparing schema body from API response with existing state.")
		equal, diags := existingState.Body.StringSemanticEquals(ctx, newBody)
		diagnostics.Append(diags...)
		if diags.HasError() {
			tflog.Warn(ctx, fmt.Sprintf("Diagnostics error while checking semantic equality of schema body: %v", diags.Errors()))
			return
		}

		if equal {
			tflog.Info(ctx, "Schema body from state and API are semantically equal. Preserving the existing state.")
			newData.Body = existingState.Body
			return
		}
	}

	tflog.Info(ctx, "Setting new schema body as state and API are not semantically equal or no existing state is present.")
	newData.Body = newBody
}

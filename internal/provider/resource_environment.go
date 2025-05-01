package provider

import (
	webclient "axual-webclient"
	custom_validator "axual.com/terraform-provider-axual/internal/custom-validator"
	"axual.com/terraform-provider-axual/internal/provider/utils"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
	"strings"
)

var _ resource.Resource = &environmentResource{}
var _ resource.ResourceWithImportState = &environmentResource{}

func NewEnvironmentResource(provider AxualProvider) resource.Resource {
	return &environmentResource{
		provider: provider,
	}
}

type environmentResource struct {
	provider AxualProvider
}

type environmentResourceData struct {
	Name                types.String `tfsdk:"name"`
	ShortName           types.String `tfsdk:"short_name"`
	Description         types.String `tfsdk:"description"`
	Color               types.String `tfsdk:"color"`
	AuthorizationIssuer types.String `tfsdk:"authorization_issuer"`
	Visibility          types.String `tfsdk:"visibility"`
	Owners              types.String `tfsdk:"owners"`
	Viewers             types.Set    `tfsdk:"viewers"`
	RetentionTime       types.Int64  `tfsdk:"retention_time"`
	Instance            types.String `tfsdk:"instance"`
	Id                  types.String `tfsdk:"id"`
	Partitions          types.Int64  `tfsdk:"partitions"`
	Properties          types.Map    `tfsdk:"properties"`
	Settings            types.Map    `tfsdk:"settings"`
}

func (r *environmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environments are used typically to support the application lifecycle, as it is moving from Development to Production.  In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2025.1/self-service/environment-management.html#managing-environments",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A suitable name identifying this environment. Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`, but not as the first character.)",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 50),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`), "can only contain letters, numbers, dots, dashes and underscores and cannot begin with an underscore, dot or dash"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "A short name that will uniquely identify this environment. The short name should be between 1 and 20 characters. Only alphanumeric characters are allowed.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 20),
					stringvalidator.RegexMatches(regexp.MustCompile((`(?i)^[a-z][a-z0-9]*$`)), "can only contain letters, numbers"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A text describing the purpose of the environment. Description must be between 1 and 200 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color used display the environment. Only these colors are allowed: `#80affe`, `#4686f0`, `#3347e1`, `#1a2dbc`, `#fee492`, `#fbd04e`, `#c2a7f9`, `#8b58f3`,`#e9b105`, `#d19e02`, `#6bdde0`, `#21ccd2`, `#19b9be`, `#069499`, `#532cd`, `#3b0d98`",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("#80affe", "#4686f0", "#3347e1", "#1a2dbc", "#fee492", "#fbd04e", "#c2a7f9", "#8b58f3",
						"#e9b105", "#d19e02", "#6bdde0", "#21ccd2", "#19b9be", "#069499", "#532cd", "#3b0d98",
					),
				}},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Can be `Public` or `Private`. The Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("Public", "Private"),
				},
			},
			"authorization_issuer": schema.StringAttribute{
				MarkdownDescription: "Allowed values: `Stream owner` and `Auto`. This indicates if any deployments on this environment should be AUTO approved or requires approval from Stream Owner. For private environments, only AUTO can be selected.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("Stream owner", "Auto"),
				},
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The id of the team owning this environment.",
				Required:            true,
			},
			"viewers": schema.SetAttribute{
				MarkdownDescription: "Environment Viewer Groups define which Groups are authorized to view all Topic Configurations and Application Authentications within the Environment, regardless of ownership and visibility. Read more: https://docs.axual.io/axual/2025.1/self-service/user-group-management.html#viewer-groups",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					custom_validator.NewNonEmptySetValidator(),
				},
			},
			"instance": schema.StringAttribute{
				MarkdownDescription: "The id of the instance where this environment should be deployed.",
				Required:            true,
			},
			"retention_time": schema.Int64Attribute{
				MarkdownDescription: "The time in milliseconds after which the messages can be deleted from all topics. If not specified, default value is 7 days (604800000). Value must be between 1000 and 160704000000 (ms).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1000),
					int64validator.AtMost(160704000000),
				},
			},
			"partitions": schema.Int64Attribute{
				MarkdownDescription: "Defines the number of partitions configured for every topic of this tenant. If not specified, default value is 2. Value must be between 1 and 120000",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(120000),
				},
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "Environment-wide properties for all topics and applications.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"settings": schema.MapAttribute{
				MarkdownDescription: "A list of Environment specific settings in Key,Value format. The options are: `enforceDataMasking`(boolean). Please note that setting `enforceDataMasking` to `true` only works if Data Masking is enabled in Tenant settings.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("enforceDataMasking")),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data environmentResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	environmentRequest, err := createEnvironmentRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elements() {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	environmentRequest.Properties = properties

	settings := make(map[string]interface{})
	for key, value := range data.Settings.Elements() {
		settings[key] = strings.Trim(value.String(), "\"")
	}
	environmentRequest.Settings = settings

	tflog.Info(ctx, fmt.Sprintf("Create environment request %q", environmentRequest))
	environment, err := r.provider.client.CreateEnvironment(environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapEnvironmentResponseToData(ctx, &data, environment)
	tflog.Trace(ctx, "Created a environment resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data environmentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := r.provider.client.GetEnvironment(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Environment not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("READ request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapEnvironmentResponseToData(ctx, &data, environment)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data environmentResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	environmentRequest, err := createEnvironmentRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	environmentRequest.Properties = r.processProperties(ctx, req, data)
	environmentRequest.Settings = r.processSettings(ctx, req, data)

	tflog.Info(ctx, fmt.Sprintf("Update environment request %q", environmentRequest))
	environment, err := r.provider.client.UpdateEnvironment(data.Id.ValueString(), environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError("UPDATE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapEnvironmentResponseToData(ctx, &data, environment)
	tflog.Trace(ctx, "Updated an environment resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data environmentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteEnvironment(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createEnvironmentRequestFromData(ctx context.Context, data *environmentResourceData, r *environmentResource) (webclient.EnvironmentRequest, error) {
	rawOwners, err := data.Owners.ToTerraformValue(ctx)
	if err != nil {
		return webclient.EnvironmentRequest{}, err
	}
	var owners string
	err = rawOwners.As(&owners)
	if err != nil {
		return webclient.EnvironmentRequest{}, err
	}
	owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)
	instance := fmt.Sprintf("%s/instances/%v", r.provider.client.ApiURL, data.Instance.ValueString())

	viewers := []string{}
	if !data.Viewers.IsNull() {
		var viewerUIDs []string
		diags := data.Viewers.ElementsAs(ctx, &viewerUIDs, false)
		if diags.HasError() {
			return webclient.EnvironmentRequest{}, fmt.Errorf("failed to extract viewers: %v", diags)
		}

		for _, viewer := range viewerUIDs {
			fullURL := fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, viewer)
			viewers = append(viewers, fullURL)
		}
	}

	environmentRequest := webclient.EnvironmentRequest{
		Name:                data.Name.ValueString(),
		ShortName:           data.ShortName.ValueString(),
		Description:         data.Description.ValueString(),
		Color:               data.Color.ValueString(),
		AuthorizationIssuer: data.AuthorizationIssuer.ValueString(),
		Visibility:          data.Visibility.ValueString(),
		Owners:              owners,
		Viewers:             viewers,
		Instance:            instance,
		RetentionTime:       int(data.RetentionTime.ValueInt64()),
		Partitions:          int(data.Partitions.ValueInt64()),
	}

	return environmentRequest, nil
}

func (r *environmentResource) processProperties(ctx context.Context, req resource.UpdateRequest, data environmentResourceData) map[string]interface{} {
	var oldPropertiesState map[string]string
	req.State.GetAttribute(ctx, path.Root("properties"), &oldPropertiesState)

	properties := make(map[string]interface{})
	// Send `properties = nil` to API if in configuration `properties = nil`, `properties = {}` or NO properties
	if data.Properties.IsNull() || data.Properties.IsUnknown() || len(data.Properties.Elements()) == 0 {
		return nil
	}

	// Mark old properties as nil
	for key := range oldPropertiesState {
		properties[key] = nil
	}
	// Add new properties
	for key, value := range data.Properties.Elements() {
		properties[key] = strings.Trim(value.String(), "\"")
	}

	return properties
}

func (r *environmentResource) processSettings(ctx context.Context, req resource.UpdateRequest, data environmentResourceData) map[string]interface{} {
	var oldSettingsState map[string]string
	req.State.GetAttribute(ctx, path.Root("settings"), &oldSettingsState)

	settings := make(map[string]interface{})
	// Send `settings = {}` to API if user defines `settings = nil`, `settings = {}` or NO settings
	if data.Settings.IsNull() || data.Settings.IsUnknown() || len(data.Settings.Elements()) == 0 {
		return make(map[string]interface{})
	}

	// Mark old settings as nil
	for key := range oldSettingsState {
		settings[key] = nil
	}
	// Add new settings
	for key, value := range data.Settings.Elements() {
		settings[key] = strings.Trim(value.String(), "\"")
	}

	return settings
}

func mapEnvironmentResponseToData(ctx context.Context, data *environmentResourceData, environment *webclient.EnvironmentResponse) {
	data.Id = types.StringValue(environment.Uid)
	data.Name = types.StringValue(environment.Name)
	data.ShortName = types.StringValue(environment.ShortName)
	data.Description = types.StringValue(environment.Description)
	data.Color = types.StringValue(environment.Color)
	data.Visibility = types.StringValue(environment.Visibility)
	data.AuthorizationIssuer = types.StringValue(environment.AuthorizationIssuer)
	data.Owners = types.StringValue(environment.Embedded.Owners.Uid)
	data.RetentionTime = types.Int64Value(int64(environment.RetentionTime))
	data.Partitions = types.Int64Value(int64(environment.Partitions))
	data.Properties = utils.HandlePropertiesMapping(ctx, data.Properties, environment.Properties)
	data.Settings = utils.HandlePropertiesMapping(ctx, data.Settings, environment.Settings)
	data.Instance = types.StringValue(environment.Embedded.Instance.Uid)

	// optional fields
	if environment.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(environment.Description)
	}

	if environment.Embedded.Viewers == nil || len(environment.Embedded.Viewers) == 0 {
		data.Viewers = types.SetNull(types.StringType)
	} else {
		viewerSet := make([]attr.Value, len(environment.Embedded.Viewers))
		for i, viewer := range environment.Embedded.Viewers {
			viewerSet[i] = types.StringValue(viewer.Uid)
		}
		viewers, diags := types.SetValue(types.StringType, viewerSet)
		if diags.HasError() {
			// Convert diagnostics to a map[string]interface{} expected by tflog.Error
			errorDetails := map[string]interface{}{
				"diagnostics": diags.Errors(),
			}
			tflog.Error(ctx, "Error creating viewers set", errorDetails)
		}
		data.Viewers = viewers
	}
}

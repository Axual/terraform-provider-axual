package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
)

var _ resource.Resource = &applicationResource{}
var _ resource.ResourceWithImportState = &applicationResource{}

func NewApplicationResource(provider AxualProvider) resource.Resource {
	return &applicationResource{
		provider: provider,
	}
}

type applicationResource struct {
	provider AxualProvider
}

type ApplicationResourceData struct {
	Name             types.String `tfsdk:"name"`
	ShortName        types.String `tfsdk:"short_name"`
	Description      types.String `tfsdk:"description"`
	ApplicationType  types.String `tfsdk:"application_type"`
	ApplicationClass types.String `tfsdk:"application_class"`
	ApplicationId    types.String `tfsdk:"application_id"`
	Type             types.String `tfsdk:"type"`
	Owners           types.String `tfsdk:"owners"`
	Visibility       types.String `tfsdk:"visibility"`
	Id               types.String `tfsdk:"id"`
}

func (r *applicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An application is responsible for producing and/or consuming data on a topic, whether it is a Java or .NET app or a connector.",
		Attributes: map[string]schema.Attribute{
			"application_type": schema.StringAttribute{
				MarkdownDescription: "Axual Application type. Possible values are Custom or Connector.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("Custom", "Connector"),
				},
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable). Read more: https://docs.axual.io/axual/2024.1/self-service/application-management.html#app-id",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Application. Must be unique. Only the special characters _, -, and . are valid as part of an application name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`), "can only contain letters, numbers, dots, dashes and underscores and cannot begin with an underscore, dot or dash"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Application short name. Unique human-readable name for the application. Only Alphanumeric and underscore allowed. Must be unique",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 60),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_]*$`), "can only contain letters, numbers, and underscores and cannot begin with an underscore"),
				},
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "Application Owner",
				Required:            true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "If application_type is Custom, type can be: Java, Pega, SAP, DotNet, Bridge. If application_type is Connector, type can be: SINK, SOURCE",
				Validators: []validator.String{
					stringvalidator.OneOf("Java", "Pega", "SAP", "DotNet", "Bridge", "SINK", "SOURCE"),
				},
			},
			"application_class": schema.StringAttribute{
				MarkdownDescription: "The application's plugin class. Required if application_type is Connector. For example com.couchbase.connect.kafka.CouchbaseSinkConnector. All available application plugin class names, pluginTypes and pluginConfigs listed here- GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application Visibility. Defines the visibility of this application. Possible values are Public and Private. Set the visibility to “Private” if you don’t want your application to end up in overviews such as the topic graph. Read more: https://docs.axual.io/axual/2024.1/self-service/application-management.html#app-visibility",
				Validators: []validator.String{
					stringvalidator.OneOf("Public", "Private"),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Application Description. A short summary describing the application",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationRequest, err := createApplicationRequestFromData(ctx, &data, *r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for application resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	Application, err := r.provider.client.CreateApplication(ApplicationRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapApplicationResponseToData(ctx, &data, Application)
	tflog.Info(ctx, "created Application")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	Application, err := r.provider.client.GetApplication(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Application, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "During READ, mapping the resource")
	mapApplicationResponseToData(ctx, &data, Application)

	tflog.Info(ctx, "During READ, saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	tflog.Error(ctx, fmt.Sprintf("Application  %q", data))

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationRequest, err := createApplicationRequestFromData(ctx, &data, *r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for application resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	Application, err := r.provider.client.UpdateApplication(data.Id.ValueString(), ApplicationRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Application, got error: %s", err))
		return
	}

	mapApplicationResponseToData(ctx, &data, Application)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteApplication(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application, got error: %s", err))
		return
	}
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createApplicationRequestFromData(ctx context.Context, data *ApplicationResourceData, r applicationResource) (webclient.ApplicationRequest, error) {
	// mandatory fields
	rawOwners, err := data.Owners.ToTerraformValue(ctx)
	if err != nil {
		return webclient.ApplicationRequest{}, err
	}
	var owners string
	err = rawOwners.As(&owners)
	if err != nil {
		return webclient.ApplicationRequest{}, err
	}
	owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)
	ApplicationRequest := webclient.ApplicationRequest{
		Name:            data.Name.ValueString(),
		ApplicationType: data.ApplicationType.ValueString(),
		ApplicationId:   data.ApplicationId.ValueString(),
		ShortName:       data.ShortName.ValueString(),
		Owners:          owners,
		Type:            data.Type.ValueString(),
		Visibility:      data.Visibility.ValueString(),
	}

	// optional fields
	if !data.Description.IsNull() {
		ApplicationRequest.Description = data.Description.ValueString()
	}

	if !data.ApplicationClass.IsNull() {
		ApplicationRequest.ApplicationClass = data.ApplicationClass.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully created Application request %q", ApplicationRequest))
	return ApplicationRequest, nil
}

func mapApplicationResponseToData(_ context.Context, data *ApplicationResourceData, application *webclient.ApplicationResponse) {
	data.Id = types.StringValue(application.Uid)
	data.ApplicationType = types.StringValue(application.ApplicationType)
	data.ApplicationId = types.StringValue(application.ApplicationId)
	data.Name = types.StringValue(application.Name)
	data.ShortName = types.StringValue(application.ShortName)
	owners := types.StringValue(application.Owners.Uid)
	data.Owners = types.StringValue(owners.ValueString())
	data.Type = types.StringValue(application.Type)
	data.Visibility = types.StringValue(application.Visibility)

	// optional fields
	if application.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(application.Description)
	}
	if application.ApplicationClass == "" {
		data.ApplicationClass = types.StringNull()
	} else {
		data.ApplicationClass = types.StringValue(application.ApplicationClass)
	}
}

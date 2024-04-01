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

var _ tfsdk.ResourceType = applicationResourceType{}
var _ tfsdk.Resource = applicationResource{}
var _ tfsdk.ResourceWithImportState = applicationResource{}

type applicationResourceType struct{}

func (t applicationResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An application represents anything that is responsible for producing and/or consuming data on a topic, whether it is a Java or .NET app or a connector.",

		Attributes: map[string]tfsdk.Attribute{
			"application_type": {
				MarkdownDescription: "Axual Application type. Possible values are Custom or Connector.",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Custom", "Connector"}),
				},
			},
			"application_id": {
				MarkdownDescription: "The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#app-id",
				Required:            true,
				Type:                types.StringType,
			},
			"name": {
				MarkdownDescription: "The name of the Application. Must be unique. Only the special characters _ , - and . are valid as part of an application name",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 50),
					validation.RegexpMatch(`^([A-Za-z0-9._-])*$`),
					validation.RegexpNotMatch(`^([-_.]).+`),
				},
			},
			"short_name": {
				MarkdownDescription: "Application short name. Unique human-readable name for the application. Only Alphanumeric and underscore allowed. Must be unique",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 60),
					validation.RegexpMatch(`^([\w])*$`),
					validation.RegexpNotMatch(`^([_]).+`),
				},
			},
			"owners": {
				MarkdownDescription: "Application Owner",
				Required:            true,
				Type:                types.StringType,
			},
			"type": {
				Required:            true,
				MarkdownDescription: "If application_type is Custom, type can be: Java, Pega, SAP, DotNet, Bridge. If application_type is Connector, type can be: SINK, SOURCE",
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Java", "Pega", "SAP", "DotNet", "Bridge", "SINK", "SOURCE"}),
				},
			},
			"application_class": {
				MarkdownDescription: "The application's plugin class. Required if application_type is Connector. For example com.couchbase.connect.kafka.CouchbaseSinkConnector. All available application plugin class names, pluginTypes and pluginConfigs listed here- GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html",
				Optional:            true,
				Type:                types.StringType,
			},
			"visibility": {
				Required:            true,
				MarkdownDescription: "Application Visibility. Defines the visibility of this application. Possible values are Public and Private. Set the visibility to “Private” if you don’t want your application to end up in overviews such as the topic graph. Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#app-visibility",
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Public", "Private"}),
				},
			},
			"description": {
				Optional:            true,
				MarkdownDescription: "Application Description. A short summary describing the application",
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(0, 200),
				},
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
				Validators: []tfsdk.AttributeValidator{
					validation.RegexpMatch(`^[0-9a-fA-F]{32}$`),
				},
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (t applicationResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationResource{
		provider: provider,
	}, diags
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

type applicationResource struct {
	provider provider
}

func (r applicationResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data ApplicationResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationRequest, err := createApplicationRequestFromData(ctx, &data, r)
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

func (r applicationResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data ApplicationResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	Application, err := r.provider.client.GetApplication(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Application not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Application, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapApplicationResponseToData(ctx, &data, Application)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data ApplicationResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	tflog.Error(ctx, fmt.Sprintf("Application  %q", data))

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationRequest, err := createApplicationRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for application resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	Application, err := r.provider.client.UpdateApplication(data.Id.Value, ApplicationRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Application, got error: %s", err))
		return
	}

	mapApplicationResponseToData(ctx, &data, Application)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data ApplicationResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteApplication(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application, got error: %s", err))
		return
	}
}

func (r applicationResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func mapApplicationResponseToData(_ context.Context, data *ApplicationResourceData, application *webclient.ApplicationResponse) {
	data.Id = types.String{Value: application.Uid}
	data.ApplicationType = types.String{Value: application.ApplicationType}
	data.ApplicationId = types.String{Value: application.ApplicationId}
	data.Name = types.String{Value: application.Name}
	data.ShortName = types.String{Value: application.ShortName}
	owners := types.String{Value: application.Owners.Uid}
	data.Owners = types.String{Value: owners.Value}
	data.Type = types.String{Value: application.Type}
	data.Visibility = types.String{Value: application.Visibility}

	// optional fields
	if application.Description == "" {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: application.Description}
	}
	if application.ApplicationClass == "" {
		data.ApplicationClass = types.String{Null: true}
	} else {
		data.ApplicationClass = types.String{Value: application.ApplicationClass}
	}
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
		Name:            data.Name.Value,
		ApplicationType: data.ApplicationType.Value,
		ApplicationId:   data.ApplicationId.Value,
		ShortName:       data.ShortName.Value,
		Owners:          owners,
		Type:            data.Type.Value,
		Visibility:      data.Visibility.Value,
	}

	// optional fields
	if !data.Description.Null {
		ApplicationRequest.Description = data.Description.Value
	}

	if !data.ApplicationClass.Null {
		ApplicationRequest.ApplicationClass = data.ApplicationClass.Value
	}

	tflog.Info(ctx, fmt.Sprintf("Application request %q", ApplicationRequest))
	return ApplicationRequest, nil
}

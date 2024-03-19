package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"net/url"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = applicationDataSourceType{}
var _ tfsdk.DataSource = applicationDataSource{}

type applicationDataSourceType struct{}

func (t applicationDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An application represents anything that is responsible for producing and/or consuming data on a topic, whether it is a Java or .NET app or a connector.",

		Attributes: map[string]tfsdk.Attribute{
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
			"application_type": {
				MarkdownDescription: "Axual Application type. Possible values are Custom.",
				Computed:            true,
				Type:                types.StringType,
			},
			"application_id": {
				MarkdownDescription: "The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#app-id",
				Computed:            true,
				Type:                types.StringType,
			},
			"name": {
				MarkdownDescription: "The name of the Application. Must be unique. Only the special characters _ , - and . are valid as part of an application name",
				Required:            true,
				Type:                types.StringType,
			},
			"short_name": {
				MarkdownDescription: "Application short name. Unique human-readable name for the application. Only Alphanumeric and underscore allowed. Must be unique",
				Computed:            true,
				Type:                types.StringType,
			},
			"owners": {
				MarkdownDescription: "Application Owner",
				Computed:            true,
				Type:                types.StringType,
			},
			"type": {
				Computed:            true,
				MarkdownDescription: "Application software. Possible values: Java, Pega, SAP, DotNet, Bridge",
				Type:                types.StringType,
			},
			"visibility": {
				Computed:            true,
				MarkdownDescription: "Application Visibility. Defines the visibility of this application. Possible values are Public and Private. Set the visibility to “Private” if you don’t want your application to end up in overviews such as the topic graph. Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#app-visibility",
				Type:                types.StringType,
			},
			"description": {
				Computed:            true,
				MarkdownDescription: "Application Description. A short summary describing the application",
				Type:                types.StringType,
			},
		},
	}, nil
}

func (t applicationDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationDataSource{
		provider: provider,
	}, diags
}

type applicationDataSourceData struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ShortName       types.String `tfsdk:"short_name"`
	Description     types.String `tfsdk:"description"`
	ApplicationType types.String `tfsdk:"application_type"`
	ApplicationId   types.String `tfsdk:"application_id"`
	Type            types.String `tfsdk:"type"`
	Owners          types.String `tfsdk:"owners"`
	Visibility      types.String `tfsdk:"visibility"`
}

type applicationDataSource struct {
	provider provider
}

func (d applicationDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data applicationDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	attributes := url.Values{}
	attributes.Set("name", data.Name.Value)
	appByName, err := d.provider.client.GetApplicationsByAttributes(attributes)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application by name, got error: %s", err))
		return
	}
	if(len(appByName.Embedded.Applications)==0) {
		resp.Diagnostics.AddError("Client Error", "Application not found")
		return 
	}
	app, err := d.provider.client.GetApplication(appByName.Embedded.Applications[0].Uid)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application, got error: %s", err))
		return
	}

	mapApplicationDataSourceResponseToData(ctx, &data, app)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapApplicationDataSourceResponseToData(ctx context.Context, data *applicationDataSourceData, app *webclient.ApplicationResponse) {

	data.Id = types.String{Value: app.Uid}
	data.ApplicationType = types.String{Value: app.ApplicationType}
	data.ApplicationId = types.String{Value: app.ApplicationId}
	data.Name = types.String{Value: app.Name}
	data.ShortName = types.String{Value: app.ShortName}
	owners := types.String{Value: app.Owners.Uid}
	data.Owners = types.String{Value: owners.Value}
	data.Type = types.String{Value: app.Type}
	data.Visibility = types.String{Value: app.Visibility}

	// optional fields
	if app.Description == "" {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: app.Description}
	}
}

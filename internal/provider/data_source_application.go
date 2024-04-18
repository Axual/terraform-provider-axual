package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &applicationDataSource{}

func NewApplicationDataSource(provider AxualProvider) datasource.DataSource {
	return &applicationDataSource{
		provider: provider,
	}
}

type applicationDataSource struct {
	provider AxualProvider
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

func (d *applicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *applicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An application is responsible for producing and/or consuming data on a topic, whether it is a Java or .NET app or a connector.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Application's unique identifier",
				Computed:            true,
			},
			"application_type": schema.StringAttribute{
				MarkdownDescription: "Axual Application type. Possible values are Custom.",
				Computed:            true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable). Read more: https://docs.axual.io/axual/2024.1/self-service/application-management.html#app-id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Application. Must be unique. Only the special characters _ , - and . are valid as part of an application name",
				Required:            true,
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Application short name. Unique human-readable name for the application. Only Alphanumeric and underscore allowed. Must be unique",
				Computed:            true,
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "Application Owner",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application software. Possible values: Java, Pega, SAP, DotNet, Bridge",
			},
			"visibility": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Visibility. Defines the visibility of this application. Possible values are Public and Private. Set the visibility to “Private” if you don’t want your application to end up in overviews such as the topic graph. Read more: https://docs.axual.io/axual/2024.1/self-service/application-management.html#app-visibility",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Description. A short summary describing the application",
			},
		},
	}
}

func (d *applicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data applicationDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	attributes := url.Values{}
	attributes.Set("name", data.Name.ValueString())
	appByName, err := d.provider.client.GetApplicationsByAttributes(attributes)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application by name, got error: %s", err))
		return
	}
	if len(appByName.Embedded.Applications) == 0 {
		resp.Diagnostics.AddError("Client Error", "Application not found")
		return
	}
	app, err := d.provider.client.GetApplication(appByName.Embedded.Applications[0].Uid)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application, got error: %s", err))
		return
	}

	mapApplicationDataSourceResponseToData(&data, app)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapApplicationDataSourceResponseToData(data *applicationDataSourceData, app *webclient.ApplicationResponse) {
	data.Id = types.StringValue(app.Uid)
	data.ApplicationType = types.StringValue(app.ApplicationType)
	data.ApplicationId = types.StringValue(app.ApplicationId)
	data.Name = types.StringValue(app.Name)
	data.ShortName = types.StringValue(app.ShortName)
	owners := types.StringValue(app.Owners.Uid)
	data.Owners = types.StringValue(owners.ValueString())
	data.Type = types.StringValue(app.Type)
	data.Visibility = types.StringValue(app.Visibility)

	// optional fields
	if app.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(app.Description)
	}
}

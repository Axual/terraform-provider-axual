package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/url"
	"regexp"
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
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ShortName        types.String `tfsdk:"short_name"`
	Description      types.String `tfsdk:"description"`
	ApplicationType  types.String `tfsdk:"application_type"`
	ApplicationClass types.String `tfsdk:"application_class"`
	ApplicationId    types.String `tfsdk:"application_id"`
	Type             types.String `tfsdk:"type"`
	Owners           types.String `tfsdk:"owners"`
	Visibility       types.String `tfsdk:"visibility"`
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
				MarkdownDescription: "Axual Application type. Possible values are Custom or Connector.",
				Computed:            true,
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable). Read more: https://docs.axual.io/axual/2025.3/self-service/application-management.html#app-id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Application. Must be unique. Only the special characters _ , - and . are valid as part of an application name",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z0-9._\- ]+$`), "can contain letters, numbers, dots, spaces, dashes and underscores"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Application short name. Unique human-readable name for the application. Only Alphanumeric and underscore allowed. Must be unique",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 60),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z0-9_]+$`), "can only contain letters, numbers and underscores and cannot begin with an underscore"),
				},
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "Application Owner",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "If application_type is Custom, type can be: Java, Pega, SAP, DotNet, Bridge. If application_type is Connector, type can be: SINK, SOURCE",
			},
			"application_class": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The application's plugin class. Required if application_type is Connector. For example com.couchbase.connect.kafka.CouchbaseSinkConnector. All available application plugin class names, pluginTypes and pluginConfigs listed here- GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html",
			},
			"visibility": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Visibility. Defines the visibility of this application. Possible values are Public and Private. Set the visibility to “Private” if you don’t want your application to end up in overviews such as the topic graph. Read more: https://docs.axual.io/axual/2025.3/self-service/application-management.html#app-visibility",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Description. A short summary describing the application",
			},
		},
	}
}

var (
	_ datasource.DataSource                     = &applicationDataSource{}
	_ datasource.DataSourceWithConfigValidators = &applicationDataSource{}
)

func (d *applicationDataSource) ConfigValidators(
	ctx context.Context,
) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf( // fail if both are null/unknown
			path.MatchRoot("name"),
			path.MatchRoot("short_name"),
		),
	}
}

func (d *applicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data applicationDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	params := url.Values{}
	searchParam := "name"
	var searchValue string

	if data.ShortName.ValueString() == "" {
		searchValue = data.Name.ValueString()
		params.Set("name", searchValue)
	} else {
		searchValue = data.ShortName.ValueString()
		params.Set("shortName", searchValue)
		searchParam = "shortName"
	}

	appResponse, err := d.provider.client.GetApplicationByNameOrShortName(params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application by %s: '%s', got error: %s", searchParam, searchValue, err))
		return
	}

	mapApplicationDataSourceResponseToData(&data, appResponse)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapApplicationDataSourceResponseToData(data *applicationDataSourceData, app *webclient.ApplicationResponse) {

	data.Id = types.StringValue(app.Uid)
	data.ApplicationType = types.StringValue(app.ApplicationType)
	data.ApplicationId = types.StringValue(app.ApplicationId)
	data.Name = types.StringValue(app.Name)
	data.ShortName = types.StringValue(app.ShortName)
	data.Owners = types.StringValue(app.Owners.Uid)
	data.Type = types.StringValue(app.Type)
	data.Visibility = types.StringValue(app.Visibility)

	// optional fields
	if app.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(app.Description)
	}

	if app.ApplicationClass == "" {
		data.ApplicationClass = types.StringNull()
	} else {
		data.ApplicationClass = types.StringValue(app.ApplicationClass)
	}
}

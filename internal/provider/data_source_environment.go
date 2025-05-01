package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &environmentDataSource{}

func NewEnvironmentDataSource(provider AxualProvider) datasource.DataSource {
	return &environmentDataSource{
		provider: provider,
	}
}

type environmentDataSource struct {
	provider AxualProvider
}

type environmentDataSourceData struct {
	Name                types.String `tfsdk:"name"`
	ShortName           types.String `tfsdk:"short_name"`
	Description         types.String `tfsdk:"description"`
	Color               types.String `tfsdk:"color"`
	AuthorizationIssuer types.String `tfsdk:"authorization_issuer"`
	Visibility          types.String `tfsdk:"visibility"`
	Owners              types.String `tfsdk:"owners"`
	RetentionTime       types.Int64  `tfsdk:"retention_time"`
	Instance            types.String `tfsdk:"instance"`
	Id                  types.String `tfsdk:"id"`
	Partitions          types.Int64  `tfsdk:"partitions"`
	Properties          types.Map    `tfsdk:"properties"`
}

func (d *environmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *environmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environments are used typically to support the application lifecycle, as it is moving from Development to Production. In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2025.1/self-service/environment-management.html#managing-environments",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A suitable name identifying this environment. Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`, but not as the first character.)",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 50),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z0-9._\- ]+$`), "can only contain letters, numbers, dots, dashes, underscores and spaces"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "A short name that will uniquely identify this environment.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 20),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z][a-z0-9]*$`), "can only contain letters, numbers and cannot begin with a number"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A text describing the purpose of the environment.",
				Computed:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "The color used to display the environment",
				Computed:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.",
				Computed:            true,
			},
			"authorization_issuer": schema.StringAttribute{
				MarkdownDescription: "This indicates if any deployments on this environment should be AUTO approved or requires approval from Topic Owner. For private environments, only AUTO can be selected.",
				Computed:            true,
			},
			"owners": schema.StringAttribute{
				MarkdownDescription: "The ID of the team owning this environment.",
				Computed:            true,
			},
			"instance": schema.StringAttribute{
				MarkdownDescription: "The ID of the instance where this environment should be deployed.",
				Computed:            true,
			},
			"retention_time": schema.Int64Attribute{
				MarkdownDescription: "The time in milliseconds after which the messages can be deleted from all topics. This is an optional field. If not specified, default value is 7 days (604800000).",
				Computed:            true,
			},

			"partitions": schema.Int64Attribute{
				MarkdownDescription: "Defines the number of partitions configured for every topic of this tenant. This is an optional field. If not specified, default value is 12",
				Computed:            true,
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "Environment-wide properties for all topics and applications.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment's unique identifier",
			},
		},
	}
}

var (
	_ datasource.DataSource                     = &environmentDataSource{}
	_ datasource.DataSourceWithConfigValidators = &environmentDataSource{}
)

func (d *environmentDataSource) ConfigValidators(
	ctx context.Context,
) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf( // fail if both are null/unknown
			path.MatchRoot("name"),
			path.MatchRoot("short_name"),
		),
	}
}

func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data environmentDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var environmentResponse *webclient.EnvironmentsResponse
	var err error

	if data.ShortName.ValueString() == "" {
		environmentResponse, err = d.provider.client.GetEnvironmentByName(data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment by name, got error: %s", err))
			return
		}
	} else {
		environmentResponse, err = d.provider.client.GetEnvironmentByShortName(data.ShortName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment by short_name, got error: %s", err))
			return
		}
	}

	// Check if Embedded or environment is nil or empty
	if len(environmentResponse.Embedded.Environments) == 0 {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("No Environment resources found with name '%s'.", data.Name.ValueString()),
		)
		return
	}

	environment, err := d.provider.client.GetEnvironment(environmentResponse.Embedded.Environments[0].Uid)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	mapEnvironmentDataSourceResponseToData(ctx, &data, environment)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapEnvironmentDataSourceResponseToData(ctx context.Context, data *environmentDataSourceData, environment *webclient.EnvironmentResponse) {
	data.Id = types.StringValue(environment.Uid)
	data.Name = types.StringValue(environment.Name)
	data.ShortName = types.StringValue(environment.ShortName)
	data.Description = types.StringValue(environment.Embedded.Instance.Description)
	data.Color = types.StringValue(environment.Color)
	data.Visibility = types.StringValue(environment.Visibility)
	data.AuthorizationIssuer = types.StringValue(environment.AuthorizationIssuer)
	data.Owners = types.StringValue(environment.Embedded.Owners.Uid)
	data.RetentionTime = types.Int64Value(int64(environment.RetentionTime))
	data.Partitions = types.Int64Value(int64(environment.Partitions))
	data.Instance = types.StringValue(environment.Links.Instance.Href)

	properties := make(map[string]attr.Value)
	for key, value := range environment.Properties {
		if value != nil {
			properties[key] = types.StringValue(value.(string))
		}
	}

	mapValue, diags := types.MapValue(types.StringType, properties)
	if diags.HasError() {
		tflog.Error(ctx, "Error creating members slice when mapping group response")
	}
	data.Properties = mapValue

	// optional fields
	if environment.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(environment.Description)
	}
}

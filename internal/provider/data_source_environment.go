package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = environmentDataSourceType{}
var _ tfsdk.DataSource = environmentDataSource{}

type environmentDataSourceType struct{}

func (t environmentDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environments are used typically to support the application lifecycle, as it is moving from Development to Production.  In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2023.2/self-service/environment-management.html#managing-environments",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "A suitable name identifying this environment. This must be in the format string-string (Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`)",
				Required:            true,
				Type:                types.StringType,
			},
			"short_name": {
				MarkdownDescription: "A short name that will uniquely identify this environment. The short name should be between 3 and 20 characters. no special characters are allowed.",
				Computed:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 50),
					validation.RegexpMatch((`^[a-z0-9]+$`)),
				},
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the environment.",
				Optional:            true,
				Computed: 			 true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(0, 200),
				},
			},
			"color": {
				MarkdownDescription: "The color used to display the environment",
				Computed:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{
						"#80affe", "#4686f0", "#3347e1", "#1a2dbc", "#fee492", "#fbd04e", "#c2a7f9", "#8b58f3",
						"#e9b105", "#d19e02", "#6bdde0", "#21ccd2", "#19b9be", "#069499", "#532cd", "#3b0d98",
					}),
				}},
			"visibility": {
				MarkdownDescription: "Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.",
				Computed:            true,
				Type:                types.StringType,
			},
			"authorization_issuer": {
				MarkdownDescription: "This indicates if any deployments on this environment should be AUTO approved or requires approval from Stream Owner. For private environments, only AUTO can be selected.",
				Computed:            true,
				Type:                types.StringType,
			},
			"owners": {
				MarkdownDescription: "The id of the team owning this environment.",
				Computed:            true,
				Type:                types.StringType,
			},
			"instance": {
				MarkdownDescription: "The id of the instance where this environment should be deployed.",
				Computed:            true,
				Type:                types.StringType,
			},
			"retention_time": {
				MarkdownDescription: "The time in milliseconds after which the messages can be deleted from all topics. This is an optional field. If not specified, default value is 7 days (604800000).",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},

			"partitions": {
				MarkdownDescription: "Defines the number of partitions configured for every topic of this tenant. This is an optional field. If not specified, default value is 12",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.GreaterThanOrEqualTo, 1),
					validation.Compare(validation.LessThanOrEqualTo, 120000),
				},
			},
			"properties": {
				MarkdownDescription: "Environment-wide properties for all topics and applications.",
				Optional:            true,
				Computed:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Environment unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t environmentDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return environmentDataSource{
		provider: provider,
	}, diags
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
	Instance             types.String `tfsdk:"instance"`
	Id                  types.String `tfsdk:"id"`
	Partitions          types.Int64  `tfsdk:"partitions"`
	Properties          types.Map    `tfsdk:"properties"`
}

type environmentDataSource struct {
	provider provider
}

func (d environmentDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data environmentDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	environmentByName, err := d.provider.client.GetEnvironmentByName(data.Name.Value)
	if err != nil {
	    resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment by short_name, got error: %s", err))
	return
	}

	environment, err := d.provider.client.GetEnvironment(environmentByName.Embedded.Environments[0].Uid)
	if err != nil {
	    resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
	return
	}

    mapEnvironmentDataSourceResponseToData(ctx, &data, environment)
	
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapEnvironmentDataSourceResponseToData(_ context.Context, data *environmentDataSourceData, environment *webclient.EnvironmentResponse) {
	data.Id = types.String{Value: environment.Uid}
	data.Name = types.String{Value: environment.Name}
	data.ShortName = types.String{Value: environment.ShortName}
	data.Description = types.String{Value: environment.Embedded.Instance.Description}
	data.Color = types.String{Value: environment.Color}
	data.Visibility = types.String{Value: environment.Visibility}
	data.AuthorizationIssuer = types.String{Value: environment.AuthorizationIssuer}
	data.Owners = types.String{Value: environment.Embedded.Owners.Uid}
	data.RetentionTime = types.Int64{Value: int64(environment.RetentionTime)}
	data.Partitions = types.Int64{Value: int64(environment.Partitions)}
	data.Instance = types.String{Value: environment.Links.Instance.Href}

	properties := make(map[string]attr.Value)
	for key, value := range environment.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}

	// optional fields
	if environment.Description == nil || len(environment.Description.(string)) == 0 {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: environment.Description.(string)}
	}
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
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

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &instanceDataSource{}

func NewInstanceDataSource(provider AxualProvider) datasource.DataSource {
	return &instanceDataSource{
		provider: provider,
	}
}

type instanceDataSource struct {
	provider AxualProvider
}

type instanceDataSourceData struct {
	Name        types.String `tfsdk:"name"`
	ShortName   types.String `tfsdk:"short_name"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
}

func (d *instanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *instanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Instance resource. Read more: https://docs.axual.io/axual/2026.1/self-service/instance-management.html",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Instance's unique identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance's name. Must be 3-50 characters long and can contain letters, numbers, dots, dashes, and underscores, but cannot start with special characters.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 50),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z0-9._\- ]+$`), "can only contain letters, numbers, dots, spaces, dashes and underscores"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Instance's short name",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 12),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^[a-z][a-z0-9]*$`), "can only contain letters and numbers, but cannot begin with a number"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Instance's description",
				Computed:            true,
			},
		},
	}
}

var (
	_ datasource.DataSource                     = &instanceDataSource{}
	_ datasource.DataSourceWithConfigValidators = &instanceDataSource{}
)

func (d *instanceDataSource) ConfigValidators(
	ctx context.Context,
) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf( // fail if both are null/unknown
			path.MatchRoot("name"),
			path.MatchRoot("short_name"),
		),
	}
}

func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data instanceDataSourceData

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

	instanceResponse, err := d.provider.client.GetInstanceByNameOrShortName(params)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError("Client Error", "Instance not found")
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance by %s: '%s', got error: %s", searchParam, searchValue, err))
		return
	}

	mapInstanceDataSourceResponseToData(&data, instanceResponse)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapInstanceDataSourceResponseToData(data *instanceDataSourceData, instance *webclient.InstanceResponse) {

	data.Id = types.StringValue(instance.Uid)
	data.Name = types.StringValue(instance.Name)
	data.ShortName = types.StringValue(instance.ShortName)
	data.Description = types.StringValue(instance.Description)

}

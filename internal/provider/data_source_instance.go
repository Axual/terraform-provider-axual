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
		MarkdownDescription: "Instance resource. Read more: https://docs.axual.io/axual/2024.4/self-service/instance-management.html",

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
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9._\- ]+$`), "can only contain letters, numbers, dots, spaces, dashes and underscores, but cannot begin with an underscore, dot, space or dash"),
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

	attributes := url.Values{}

	if data.ShortName.ValueString() == "" {
		attributes.Set("name", data.Name.ValueString())
	} else {
		attributes.Set("short_name", data.ShortName.ValueString())
	}

	instanceResponse, err := d.provider.client.GetInstancesByAttributes(attributes)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance by name, got error: %s", err))
		return
	}

	if len(instanceResponse.Embedded.Instances) == 0 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Instance not found"))
		return
	}

	mapInstanceDataSourceResponseToData(ctx, &data, instanceResponse)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapInstanceDataSourceResponseToData(ctx context.Context, data *instanceDataSourceData, instanceResponseAttributes *webclient.InstancesResponseByAttributes) {

	instance := instanceResponseAttributes.Embedded.Instances[0]

	data.Id = types.StringValue(instance.Uid)
	data.Name = types.StringValue(instance.Name)
	data.ShortName = types.StringValue(instance.ShortName)
	data.Description = types.StringValue(instance.Description)

}

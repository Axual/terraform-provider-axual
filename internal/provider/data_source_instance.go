package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
		MarkdownDescription: "Instance resource. Read more: https://docs.axual.io/axual/2024.2/self-service/instance-management.html",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Instance's unique identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance's name",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`), "can only contain letters, numbers, dots, spaces, dashes and underscores, but cannot begin with an underscore, dot, space or dash"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Instance's short name",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Instance's description",
				Computed:            true,
			},
		},
	}
}

func (d *instanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data instanceDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	instanceByName, err := d.provider.client.GetInstanceByName(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance by name, got error: %s", err))
		return
	}

	instance, err2 := d.provider.client.GetInstance(instanceByName.Uid)
	if err2 != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance, got error: %s", err2))
		return
	}

	mapInstanceDataSourceResponseToData(ctx, &data, instance)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapInstanceDataSourceResponseToData(ctx context.Context, data *instanceDataSourceData, instance *webclient.InstanceResponse) {
	data.Id = types.StringValue(instance.Uid)
	data.Name = types.StringValue(instance.Name)
	data.ShortName = types.StringValue(instance.ShortName)
	data.Description = types.StringValue(instance.Description)

}

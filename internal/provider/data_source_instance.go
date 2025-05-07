package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "Instance's short name",
				Optional:            true,
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

	attributes := url.Values{}

	validateIfNameOrShortNamePresent(data.Name.ValueString(), data.ShortName.ValueString(), resp)

	if data.ShortName.ValueString() != "" {
		validateInstanceShortName(data.ShortName.ValueString(), resp)
	}

	if data.Name.ValueString() != "" {
		validateInstanceName(data.Name.ValueString(), resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

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

	instance, err2 := d.provider.client.GetInstance(instanceResponse.Embedded.Instances[0].Uid)
	if err2 != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read instance with ID '%s', got error: %s", instanceResponse.Embedded.Instances[0].Uid, err2))
		return
	}

	mapInstanceDataSourceResponseToData(ctx, &data, instance)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func validateInstanceShortName(shortName string, resp *datasource.ReadResponse) {
	if len(shortName) < 1 || len(shortName) > 12 {
		resp.Diagnostics.AddError("Invalid ShortName Length", "ShortName must be between 1 and 12 characters")
		return
	}

	match := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`).MatchString(shortName)
	if !match {
		resp.Diagnostics.AddError("Invalid ShortName Format", "ShortName must contain letter or number but cannot begin with a number")
		return
	}
}

func validateInstanceName(name string, resp *datasource.ReadResponse) {
	if len(name) < 3 || len(name) > 50 {
		resp.Diagnostics.AddError("Invalid Name Length", "Name must be between 3 and 50 characters")
		return
	}

	match := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`).MatchString(name)
	if !match {
		resp.Diagnostics.AddError("Invalid Name Format", "Name must contain letters, numbers, dots, spaces, dashes and underscores, but cannot begin with an underscore, dot, space or dash")
		return
	}

}

func mapInstanceDataSourceResponseToData(ctx context.Context, data *instanceDataSourceData, instance *webclient.InstanceResponse) {
	data.Id = types.StringValue(instance.Uid)
	data.Name = types.StringValue(instance.Name)
	data.ShortName = types.StringValue(instance.ShortName)
	data.Description = types.StringValue(instance.Description)

}

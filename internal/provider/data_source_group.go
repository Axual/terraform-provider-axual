package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
)

var _ datasource.DataSource = &groupDataSource{}

func NewGroupDataSource(provider AxualProvider) datasource.DataSource {
	return &groupDataSource{
		provider: provider,
	}
}

type groupDataSource struct {
	provider AxualProvider
}

type groupDataSourceData struct {
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Members      types.Set    `tfsdk:"members"`
	Id           types.String `tfsdk:"id"`
}

func (d *groupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Group resource. Read more: https://docs.axual.io/axual/2024.2/self-service/user-group-management.html#groups",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Group's unique identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Group's name",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`), "can only contain letters, numbers, dots, spaces, dashes and underscores, but cannot begin with an underscore, dot, space or dash"),
				},
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Group's email address",
				Computed:            true,
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "Group's phone number",
				Computed:            true,
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "Group's members",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data groupDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupByName, err := d.provider.client.GetGroupByName(data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group by name, got error: %s", err))
		return
	}

	group, err2 := d.provider.client.GetGroup(groupByName.Embedded.Groups[0].Uid)
	if err2 != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group, got error: %s", err2))
		return
	}

	mapGroupDataSourceResponseToData(ctx, &data, group)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapGroupDataSourceResponseToData(ctx context.Context, data *groupDataSourceData, group *webclient.GroupResponse) {

	data.Id = types.StringValue(group.Uid)
	data.Name = types.StringValue(group.Name)
	var members []attr.Value
	for _, member := range group.Embedded.Members {
		members = append(members, types.StringValue(member.Uid))
	}

	setValue, diags := types.SetValue(types.StringType, members)

	if diags.HasError() {
		tflog.Error(ctx, "Error creating members slice when mapping group response")
	}

	data.Members = setValue

	// optional fields
	if group.EmailAddress == nil {
		data.EmailAddress = types.StringNull()
	} else {
		m := group.EmailAddress.(map[string]interface{})
		data.EmailAddress = types.StringValue(m["email"].(string))
	}
	if group.PhoneNumber == nil {
		data.PhoneNumber = types.StringNull()
	} else {
		data.PhoneNumber = types.StringValue(group.PhoneNumber.(string))
	}
}

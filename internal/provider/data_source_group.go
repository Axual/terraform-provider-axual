package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = groupDataSourceType{}
var _ tfsdk.DataSource = groupDataSource{}

type groupDataSourceType struct{}

func (t groupDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Group resource. Read more: https://docs.axual.io/axual/2024.1/self-service/user-group-management.html#groups",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "Group's name",
				Required:            true,
				Type:                types.StringType,
			},
			"email_address": {
				MarkdownDescription: "Group's email address",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
			},
			"phone_number": {
				MarkdownDescription: "Group's phone number",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
			},
			"members": {
				MarkdownDescription: "Group's members",
				Optional:            true,
				Computed:            true,
				Type:                types.SetType{ElemType: types.StringType},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Group's unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t groupDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return groupDataSource{
		provider: provider,
	}, diags
}

type groupDataSourceData struct {
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Members      types.Set    `tfsdk:"members"`
	Id           types.String `tfsdk:"id"`
}

type groupDataSource struct {
	provider provider
}

func (d groupDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data groupDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupByName, err := d.provider.client.GetGroupByName(data.Name.Value)
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

	data.Id = types.String{Value: group.Uid}
	data.Name = types.String{Value: group.Name}
	var members []attr.Value
	for _, member := range group.Embedded.Members {
		members = append(members, types.String{Value: member.Uid})
	}
	data.Members = types.Set{Elems: members, ElemType: types.StringType}

	// optional fields
	if group.EmailAddress == nil {
		data.EmailAddress = types.String{Null: true}
	} else {
		m := group.EmailAddress.(map[string]interface{})
		data.EmailAddress = types.String{Value: m["email"].(string)}
	}
	if group.PhoneNumber == nil {
		data.PhoneNumber = types.String{Null: true}
	} else {
		data.PhoneNumber = types.String{Value: group.PhoneNumber.(string)}
	}

}

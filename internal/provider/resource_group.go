package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = groupResourceType{}
var _ tfsdk.Resource = groupResource{}
var _ tfsdk.ResourceWithImportState = groupResource{}

type groupResourceType struct{}

func (t groupResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Group resource. Read more: https://docs.axual.io/axual/2023.2/self-service/user-group-management.html#groups",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "Group's name",
				Required:            true,
				Type:                types.StringType,
			},
			"email_address": {
				MarkdownDescription: "Group's email address",
				Optional:            true,
				Type:                types.StringType,
			},
			"phone_number": {
				MarkdownDescription: "Group's phone number",
				Optional:            true,
				Type:                types.StringType,
			},
			"members": {
				MarkdownDescription: "Group's members",
				Optional:            true,
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

func (t groupResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return groupResource{
		provider: provider,
	}, diags
}

type groupResourceData struct {
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Members      types.Set    `tfsdk:"members"`
	Id           types.String `tfsdk:"id"`
}

type groupResource struct {
	provider provider
}

func (r groupResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data groupResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupRequest, err := createGroupRequestFromData(ctx, &data, r.provider.client.ApiURL)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	group, err := r.provider.client.CreateGroup(groupRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapGroupResponseToData(ctx, &data, group)
	tflog.Trace(ctx, "created a resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r groupResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.provider.client.ReadGroup(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Group not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapGroupResponseToData(ctx, &data, group)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r groupResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data groupResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupRequest, err := createGroupRequestFromData(ctx, &data, r.provider.client.ApiURL)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	group, err := r.provider.client.UpdateGroup(data.Id.Value, groupRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update group, got error: %s", err))
		return
	}

	mapGroupResponseToData(ctx, &data, group)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r groupResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteGroup(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete group, got error: %s", err))
		return
	}
}

func (r groupResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func mapGroupResponseToData(ctx context.Context, data *groupResourceData, group *webclient.GroupResponse) {
	// mandatory fields first
	tflog.Info(ctx, "mapping response to data")
	data.Id = types.String{Value: group.Uid}
	data.Name = types.String{Value: group.Name}
	var members []attr.Value
	for _, member := range group.Embedded.Members {
		members = append(members, types.String{Value: member.Uid})
	}
	data.Members = types.Set{Elems: members, ElemType: types.StringType}

	// optional fields
	if nil == group.EmailAddress {
		data.EmailAddress = types.String{Null: true}
	} else {
		tflog.Info(ctx, fmt.Sprintf("email is %s", group.EmailAddress))
		m := group.EmailAddress.(map[string]interface{})
		data.EmailAddress = types.String{Value: m["email"].(string)}
	}
	if group.PhoneNumber == nil {
		data.PhoneNumber = types.String{Null: true}
	} else {
		data.PhoneNumber = types.String{Value: group.PhoneNumber.(string)}
	}

}

func createGroupRequestFromData(ctx context.Context, data *groupResourceData, apiUrl string) (webclient.GroupRequest, error) {
	// mandatory fields

	var members []string
	for _, raw := range data.Members.Elems {
		value, err := raw.ToTerraformValue(ctx)
		if err != nil {
			return webclient.GroupRequest{}, err
		}
		var member string
		err = value.As(&member)
		if err != nil {
			return webclient.GroupRequest{}, err
		}
		members = append(members, fmt.Sprintf("%s/users/%v", apiUrl, member))
	}
	tflog.Info(ctx, fmt.Sprintf("Desired members list size %d", len(data.Members.Elems)))
	tflog.Info(ctx, fmt.Sprintf("Creating new members list of size %d", len(members)))

	groupRequest := webclient.GroupRequest{
		Name:    data.Name.Value,
		Members: members,
	}

	// optional fields
	if !data.PhoneNumber.Null {
		tflog.Info(ctx, "phone number is not null")
		groupRequest.PhoneNumber = data.PhoneNumber.Value
	}

	if !data.EmailAddress.Null {
		tflog.Info(ctx, "email is not null")
		groupRequest.EmailAddress = data.EmailAddress.Value
	}

	tflog.Info(ctx, fmt.Sprintf("group request %q", groupRequest))
	return groupRequest, nil
}

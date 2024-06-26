package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
)

var _ resource.Resource = &groupResource{}
var _ resource.ResourceWithImportState = &groupResource{}

type groupResourceType struct{}

func NewGroupResource(provider AxualProvider) resource.Resource {
	return &groupResource{
		provider: provider,
	}
}

type groupResource struct {
	provider AxualProvider
}

type groupResourceData struct {
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Members      types.Set    `tfsdk:"members"`
	Id           types.String `tfsdk:"id"`
}

func (r *groupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Group resource. Read more: https://docs.axual.io/axual/2024.1/self-service/user-group-management.html#groups",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Group's name",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._ -]*$`), "can only contain letters, numbers, dots, dashes and underscores, but cannot begin with an underscore, dot or dash"),
				},
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Group's email address",
				Optional:            true,
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "Group's phone number",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 16),
				},
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "Group's members",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group's unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.provider.client.GetGroup(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Group not found. Id: %s", data.Id.ValueString()))
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

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	group, err := r.provider.client.UpdateGroup(data.Id.ValueString(), groupRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update group, got error: %s", err))
		return
	}

	mapGroupResponseToData(ctx, &data, group)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteGroup(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete group, got error: %s", err))
		return
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapGroupResponseToData(ctx context.Context, data *groupResourceData, group *webclient.GroupResponse) {
	// mandatory fields first
	tflog.Info(ctx, "mapping response to data")
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
	if nil == group.EmailAddress {
		data.EmailAddress = types.StringNull()
	} else {
		tflog.Info(ctx, fmt.Sprintf("email is %s", group.EmailAddress))
		m := group.EmailAddress.(map[string]interface{})
		data.EmailAddress = types.StringValue(m["email"].(string))
	}
	if group.PhoneNumber == nil {
		data.PhoneNumber = types.StringNull()
	} else {
		data.PhoneNumber = types.StringValue(group.PhoneNumber.(string))
	}

}

func createGroupRequestFromData(ctx context.Context, data *groupResourceData, apiUrl string) (webclient.GroupRequest, error) {
	// mandatory fields

	var members []string
	for _, raw := range data.Members.Elements() {
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
	tflog.Info(ctx, fmt.Sprintf("Desired members list size %d", len(data.Members.Elements())))
	tflog.Info(ctx, fmt.Sprintf("Creating new members list of size %d", len(members)))

	groupRequest := webclient.GroupRequest{
		Name:    data.Name.ValueString(),
		Members: members,
	}

	// optional fields
	if !data.PhoneNumber.IsNull() {
		tflog.Info(ctx, "phone number is not null")
		groupRequest.PhoneNumber = data.PhoneNumber.ValueString()
	}

	if !data.EmailAddress.IsNull() {
		tflog.Info(ctx, "email is not null")
		groupRequest.EmailAddress = data.EmailAddress.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("group request %q", groupRequest))
	return groupRequest, nil
}

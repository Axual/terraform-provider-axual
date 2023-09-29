package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = userResourceType{}
var _ tfsdk.Resource = userResource{}
var _ tfsdk.ResourceWithImportState = userResource{}

type userResourceType struct{}

func (t userResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "User resource. Read more: https://docs.axual.io/axual/2023.2/self-service/user-group-management.html#users",

		Attributes: map[string]tfsdk.Attribute{
			"first_name": {
				MarkdownDescription: "User's first name",
				Required:            true,
				Type:                types.StringType,
			},
			"middle_name": {
				MarkdownDescription: "User's middle name",
				Optional:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(1, -1),
				},
			},
			"last_name": {
				MarkdownDescription: "User's last name",
				Required:            true,
				Type:                types.StringType,
			},
			"email_address": {
				MarkdownDescription: "User's email address",
				Required:            true,
				Type:                types.StringType,
			},
			"phone_number": {
				MarkdownDescription: "User's phone number",
				Optional:            true,
				Type:                types.StringType,
			},
			"roles": {
				MarkdownDescription: "Roles attributed to the user. All possible roles with descriptions are listed here: https://docs.axual.io/apidocs/mgmt-api/6.12.1/index.html#valid-roles",
				Optional:            true,
				Attributes: tfsdk.SetNestedAttributes(map[string]tfsdk.Attribute{
					"name": {
						Type:     types.StringType,
						Required: true,
					},
				}),
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "User unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t userResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return userResource{
		provider: provider,
	}, diags
}

type userResourceData struct {
	FirstName    types.String `tfsdk:"first_name"`
	MiddleName   types.String `tfsdk:"middle_name"`
	LastName     types.String `tfsdk:"last_name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Roles        []Role       `tfsdk:"roles"`
	Id           types.String `tfsdk:"id"`
}

type Role struct {
	Name types.String `tfsdk:"name"`
}

type userResource struct {
	provider provider
}

func (r userResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data userResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	userRequest := createUserRequestFromData(ctx, &data)

	user, err := r.provider.client.CreateUser(userRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for user resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapUserResponseToData(ctx, &data, user)
	tflog.Trace(ctx, "created a resource")
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r userResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data userResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.provider.client.GetUser(data.Id.Value)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("User not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapUserResponseToData(ctx, &data, user)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r userResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data userResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	userRequest := createUserRequestFromData(ctx, &data)

	user, err := r.provider.client.UpdateUser(data.Id.Value, userRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user, got error: %s", err))
		return
	}

	mapUserResponseToData(ctx, &data, user)
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r userResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data userResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteUser(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user, got error: %s", err))
		return
	}
}

func (r userResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func mapUserResponseToData(_ context.Context, data *userResourceData, user *webclient.UserResponse) {
	// mandatory fields first
	data.Id = types.String{Value: user.Uid}
	data.FirstName = types.String{Value: user.FirstName}
	data.LastName = types.String{Value: user.LastName}
	data.EmailAddress = types.String{Value: user.EmailAddress.Email}
	var newRoles []Role
	for _, role := range user.Roles {
		newRoles = append(newRoles, Role{Name: types.String{Value: role.Name}})
	}
	data.Roles = newRoles

	// optional fields
	if user.PhoneNumber == nil {
		data.PhoneNumber = types.String{Null: true}
	} else {
		data.PhoneNumber = types.String{Value: user.PhoneNumber.(string)}
	}

	if user.MiddleName == nil || len(user.MiddleName.(string)) == 0 {
		data.MiddleName = types.String{Null: true}
	} else {
		data.MiddleName = types.String{Value: user.MiddleName.(string)}
	}
}

func createUserRequestFromData(ctx context.Context, data *userResourceData) webclient.UserRequest {
	// mandatory fields
	var roles []webclient.UserRole

	for _, raw := range data.Roles {
		roles = append(roles, webclient.UserRole{raw.Name.Value})
	}
	tflog.Info(ctx, fmt.Sprintf("Desired roles list size %d", len(data.Roles)))
	tflog.Info(ctx, fmt.Sprintf("Creating new roles list of size %d", len(roles)))
	userRequest := webclient.UserRequest{
		FirstName:    data.FirstName.Value,
		LastName:     data.LastName.Value,
		EmailAddress: data.EmailAddress.Value,
		Roles:        roles,
	}

	// optional fields
	if !data.PhoneNumber.Null {
		userRequest.PhoneNumber = data.PhoneNumber.Value
	}

	if !data.MiddleName.Null {
		userRequest.MiddleName = data.MiddleName.Value
	}

	tflog.Info(ctx, fmt.Sprintf("user request %q", userRequest))
	return userRequest
}

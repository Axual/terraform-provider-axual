package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &userResource{}
var _ resource.ResourceWithImportState = &userResource{}

func NewUserResource(provider AxualProvider) resource.Resource {
	return &userResource{
		provider: provider,
	}
}

type userResource struct {
	provider AxualProvider
}

type userResourceData struct {
	FirstName    types.String `tfsdk:"first_name"`
	MiddleName   types.String `tfsdk:"middle_name"`
	LastName     types.String `tfsdk:"last_name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Roles        types.Set    `tfsdk:"roles"`
	Id           types.String `tfsdk:"id"`
}

type Role struct {
	Name types.String `tfsdk:"name"`
}

// roleObjectType is the type of one element of roles.
var roleObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "User resource. Creating new users via Terraform is not supported. Use the axual_user data source to reference existing users, or use terraform import to manage existing users. New users are created by logging in through your organization's SSO provider. Read more: https://docs.axual.io/axual/2026.1/self-service/user-group-management.html#users",
		Attributes: map[string]schema.Attribute{
			"first_name": schema.StringAttribute{
				MarkdownDescription: "User's first name",
				Required:            true,
			},
			"middle_name": schema.StringAttribute{
				MarkdownDescription: "User's middle name",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "User's last name",
				Required:            true,
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "User's email address",
				Required:            true,
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "User's phone number",
				Optional:            true,
			},
			"roles": schema.SetNestedAttribute{
				MarkdownDescription: "Roles attributed to the user. Leave it out to keep the roles the user has; `roles = []` removes all roles. All possible roles with descriptions are listed here: https://docs.axual.io/apidocs/mgmt-api/8.5.0/index.html#valid-roles",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"User creation is not supported",
		"The POST /users API endpoint has been removed. "+
			"Users cannot be created via Terraform. Please use terraform import to manage existing users, "+
			"or use the axual_user data source to reference users.",
	)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data userResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.provider.client.GetUser(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("User not found. Id: %s", data.Id.ValueString()))
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

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data userResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.desiredRoles(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read the roles of user %s, got error: %s", data.Id.ValueString(), err))
		return
	}
	userRequest := createUserRequestFromData(ctx, &data, roles)

	user, err := r.provider.client.UpdateUser(data.Id.ValueString(), userRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user, got error: %s", err))
		return
	}

	mapUserResponseToData(ctx, &data, user)
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data userResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := ignoreGone(r.provider.client.DeleteUser(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user, got error: %s", err))
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importByIDOrKey(ctx, req, resp, importByEmailPrefix, func(key string) (string, error) {
		return findUserIDByEmail(r.provider.client, key)
	})
}

// desiredRoles returns the roles to send. Unknown roles (left out, while the state has none, for example
// a state from 3.2.x with -refresh=false) keep the roles the user has.
func (r *userResource) desiredRoles(ctx context.Context, data userResourceData) ([]Role, error) {
	if data.Roles.IsUnknown() || data.Roles.IsNull() {
		user, err := r.provider.client.GetUser(data.Id.ValueString())
		if err != nil {
			return nil, err
		}
		return rolesOf(user), nil
	}
	var roles []Role
	if diags := data.Roles.ElementsAs(ctx, &roles, false); diags.HasError() {
		return nil, fmt.Errorf("unable to read the planned roles: %v", diags)
	}
	return roles, nil
}

func rolesOf(user *webclient.UserResponse) []Role {
	roles := make([]Role, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, Role{Name: types.StringValue(role.Name)})
	}
	return roles
}

func mapUserResponseToData(ctx context.Context, data *userResourceData, user *webclient.UserResponse) {
	// mandatory fields first
	data.Id = types.StringValue(user.Uid)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.EmailAddress = types.StringValue(user.EmailAddress.Email)
	roles, diags := types.SetValueFrom(ctx, roleObjectType, rolesOf(user))
	if diags.HasError() {
		tflog.Error(ctx, "Error creating roles set")
	}
	data.Roles = roles

	// optional fields
	if user.PhoneNumber == nil {
		data.PhoneNumber = types.StringNull()
	} else {
		data.PhoneNumber = types.StringValue(user.PhoneNumber.(string))
	}

	if user.MiddleName == nil || len(user.MiddleName.(string)) == 0 {
		data.MiddleName = types.StringNull()
	} else {
		data.MiddleName = types.StringValue(user.MiddleName.(string))
	}
}

func createUserRequestFromData(ctx context.Context, data *userResourceData, desiredRoles []Role) webclient.UserRequest {
	// mandatory fields
	// Not nil: a nil slice is sent as JSON null, which the roles endpoint refuses; [] clears the roles.
	roles := make([]webclient.UserRole, 0, len(desiredRoles))
	for _, raw := range desiredRoles {
		roles = append(roles, webclient.UserRole{Name: raw.Name.ValueString()})
	}
	tflog.Info(ctx, fmt.Sprintf("Desired roles list size %d", len(desiredRoles)))
	tflog.Info(ctx, fmt.Sprintf("Creating new roles list of size %d", len(roles)))
	userRequest := webclient.UserRequest{
		FirstName:    data.FirstName.ValueString(),
		LastName:     data.LastName.ValueString(),
		EmailAddress: data.EmailAddress.ValueString(),
		Roles:        roles,
	}

	// optional fields
	if !data.PhoneNumber.IsNull() {
		userRequest.PhoneNumber = data.PhoneNumber.ValueString()
	}

	if !data.MiddleName.IsNull() {
		userRequest.MiddleName = data.MiddleName.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("user request %q", userRequest))
	return userRequest
}

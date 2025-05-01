package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	Roles        []Role       `tfsdk:"roles"`
	Id           types.String `tfsdk:"id"`
}

type Role struct {
	Name types.String `tfsdk:"name"`
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "User resource. Please note that creating a new user with Terraform does not automatically allow the user to log in. This is because the user is only created in the Self-Service Database, not in an authentication provider such as Keycloak or Auth0. For new users please either use user data source or import using with terraform import command. Read more about user in Axual Self-Service: https://docs.axual.io/axual/2025.1/self-service/user-group-management.html#users",
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
				MarkdownDescription: "Roles attributed to the user. All possible roles with descriptions are listed here: https://docs.axual.io/apidocs/mgmt-api/8.5.0/index.html#valid-roles",
				Optional:            true,
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

	userRequest := createUserRequestFromData(ctx, &data)

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

	err := r.provider.client.DeleteUser(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user, got error: %s", err))
		return
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapUserResponseToData(_ context.Context, data *userResourceData, user *webclient.UserResponse) {
	// mandatory fields first
	data.Id = types.StringValue(user.Uid)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.EmailAddress = types.StringValue(user.EmailAddress.Email)
	var newRoles []Role
	for _, role := range user.Roles {
		newRoles = append(newRoles, Role{Name: types.StringValue(role.Name)})
	}
	data.Roles = newRoles

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

func createUserRequestFromData(ctx context.Context, data *userResourceData) webclient.UserRequest {
	// mandatory fields
	var roles []webclient.UserRole

	for _, raw := range data.Roles {
		roles = append(roles, webclient.UserRole{raw.Name.ValueString()})
	}
	tflog.Info(ctx, fmt.Sprintf("Desired roles list size %d", len(data.Roles)))
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

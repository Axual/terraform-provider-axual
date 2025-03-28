package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &userDataSource{}

func NewUserDataSource(provider AxualProvider) datasource.DataSource {
	return &userDataSource{
		provider: provider,
	}
}

type userDataSource struct {
	provider AxualProvider
}

type userDataSourceData struct {
	Email       types.String `tfsdk:"email"`
	Id          types.String `tfsdk:"id"`
	FirstName   types.String `tfsdk:"first_name"`
	LastName    types.String `tfsdk:"last_name"`
	MiddleName  types.String `tfsdk:"middle_name"`
	PhoneNumber types.String `tfsdk:"phone_number"`
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "User resource. Retrieves user details by email address.",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				MarkdownDescription: "User's email address. Must be a valid email format.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "User's unique identifier",
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "User's first name",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "User's last name",
				Computed:            true,
			},
			"middle_name": schema.StringAttribute{
				MarkdownDescription: "User's middle name",
				Computed:            true,
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "User's phone number",
				Computed:            true,
			},
		},
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	usersResponse, err := d.provider.client.FindUserByEmail(data.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find user by email, got error: %s", err))
		return
	}

	// If no users were found, add an error to diagnostics and exit.
	if len(usersResponse.Embedded.Users) == 0 {
		resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("No user found with email '%s'. Please check the email address and try again.", data.Email.ValueString()))
		return
	}

	// Map the API response to the Terraform data structure.
	mapUserDataSourceResponseToData(ctx, &data, usersResponse)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapUserDataSourceResponseToData(ctx context.Context, data *userDataSourceData, usersResponse *webclient.UsersResponse) {
	// Since the email is unique, we assume there is always exactly one user in the response.
	user := usersResponse.Embedded.Users[0]

	data.Id = types.StringValue(user.UID)
	data.Email = types.StringValue(user.Emailaddress.Email)
	data.FirstName = types.StringValue(user.Firstname)
	data.LastName = types.StringValue(user.Lastname)

	// Map the middle name, defaulting to an empty string if nil.
	data.MiddleName = types.StringValue("")
	if user.Middlename != nil {
		if middle, ok := user.Middlename.(string); ok {
			data.MiddleName = types.StringValue(middle)
		}
	}

	// Map the phone number, defaulting to an empty string if nil.
	data.PhoneNumber = types.StringValue("")
	if user.Phonenumber != nil {
		if phone, ok := user.Phonenumber.(string); ok {
			data.PhoneNumber = types.StringValue(phone)
		}
	}
}

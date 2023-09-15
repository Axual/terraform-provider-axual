package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"log"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = environmentDataSourceType{}
var _ tfsdk.DataSource = environmentDataSource{}

type environmentDataSourceType struct{}

func (t environmentDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Environments are used typically to support the application lifecycle, as it is moving from Development to Production.  In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2023.2/self-service/environment-management.html#managing-environments",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "A suitable name identifying this environment. This must be in the format string-string (Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`)",
				Computed:            true,
				Type:                types.StringType,
			},
			"short_name": {
				MarkdownDescription: "A short name that will uniquely identify this environment. The short name should be between 3 and 20 characters. no special characters are allowed.",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 50),
					validation.RegexpMatch((`^[a-z0-9]+$`)),
				},
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the environment.",
				Computed:            true,
				Type:                types.StringType,
			},
			"color": {
				MarkdownDescription: "The color used display the environment",
				Computed:            true,
				Type:                types.StringType,
				},
			"visibility": {
				MarkdownDescription: "Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.",
				Computed:            true,
				Type:                types.StringType,
			},
			"created_at": {
				MarkdownDescription: "The created at time",
				Computed:            true,
				Type:                types.StringType,
			},
			"created_by": {
				MarkdownDescription: "The email of the user that created the environment",
				Computed:            true,
				Type:                types.StringType,
			},
			"modified_at": {
				MarkdownDescription: "The last modified at time",
				Computed:            true,
				Type:                types.StringType,
			},
			"modified_by": {
				MarkdownDescription: "The email of the user that last modified the environment",
				Computed:            true,
				Type:                types.StringType,
			},
			"owners": {
				MarkdownDescription: "The link of the team owning this environment.",
				Computed:            true,
				Type:                types.StringType,
			},
			"instance": {
				MarkdownDescription: "The link of the instance where this environment should be deployed.",
				Computed:            true,
				Type:                types.StringType,
			},
			
			"id": {
				Computed:            true,
				MarkdownDescription: "Environment unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t environmentDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return environmentDataSource{
		provider: provider,
	}, diags
}

type environmentDataSourceData struct {
	Name                types.String `tfsdk:"name"`
	ShortName           types.String `tfsdk:"short_name"`
	Description         types.String `tfsdk:"description"`
	Color               types.String `tfsdk:"color"`
	Visibility          types.String `tfsdk:"visibility"`
	Owners              types.String `tfsdk:"owners"`
	Instnce             types.String `tfsdk:"instance"`
	Id                  types.String `tfsdk:"id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	CreatedBy           types.String `tfsdk:"created_by"`
	ModifiedAt           types.String `tfsdk:"modified_at"`
	ModifiedBy           types.String `tfsdk:"modified_by"`
}

type environmentDataSource struct {
	provider provider
}

func (d environmentDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data environmentDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	log.Printf("got here")

	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("got here")


	environment, err := d.provider.client.ReadEnvironmentFromDataSource(data.ShortName.Value)
	if err != nil {
		
	    resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
	    return
	}

    mapEnvironmentDataSourceResponseToData(ctx, &data, environment)
	
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapEnvironmentDataSourceResponseToData(_ context.Context, data *environmentDataSourceData, environment *webclient.EnvironmentsResponse) {
	data.Id = types.String{Value: environment.Embedded.Environments[0].Uid}
	data.Name = types.String{Value: environment.Embedded.Environments[0].Name}
	data.ShortName = types.String{Value: environment.Embedded.Environments[0].ShortName}
	data.Description = types.String{Value: environment.Embedded.Environments[0].Description}
	data.Color = types.String{Value: environment.Embedded.Environments[0].Color}
	data.Visibility = types.String{Value: environment.Embedded.Environments[0].Visibility}
	data.CreatedAt = types.String{Value: environment.Embedded.Environments[0].CreatedAt}
	data.CreatedBy = types.String{Value: environment.Embedded.Environments[0].CreatedBy}
	data.ModifiedAt = types.String{Value: environment.Embedded.Environments[0].ModifiedAt}
	data.ModifiedBy = types.String{Value: environment.Embedded.Environments[0].ModifiedBy}
    data.Id = types.String{Value: environment.Embedded.Environments[0].Uid}
	data.Owners = types.String{Value: environment.Embedded.Environments[0].Links.Owners.Href}
	data.Instnce = types.String{Value: environment.Embedded.Environments[0].Links.Instance.Href}
}

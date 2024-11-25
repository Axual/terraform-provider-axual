package provider

import (
	webclient "axual-webclient"
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &AxualProvider{}
var _ provider.ProviderWithFunctions = &AxualProvider{}

type AxualProvider struct {
	// client can contain the upstream provider SDK or HTTP client used to
	// communicate with the upstream service. Resource and DataSource
	// implementations can then make calls using this client.
	client *webclient.Client

	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// providerData can be used to store data from the Terraform configuration.
type providerData struct {
	ApiUrl   types.String `tfsdk:"apiurl"`
	Realm    types.String `tfsdk:"realm"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	ClientID types.String `tfsdk:"clientid"`
	AuthUrl  types.String `tfsdk:"authurl"`
	Scopes   types.List   `tfsdk:"scopes"`
}

func (p *AxualProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiurl := data.ApiUrl.ValueString()
	realm := data.Realm.ValueString()

	var username string
	if data.Username.IsNull() {
		username = os.Getenv("AXUAL_AUTH_USERNAME")
		if username == "" {
			resp.Diagnostics.AddError(
				"Missing Username",
				"Username is not provided in configuration and the AXUAL_AUTH_USERNAME environment variable is not set.",
			)
			return
		}
	} else {
		username = data.Username.ValueString()
	}

	var password string
	if data.Password.IsUnknown() {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as host",
		)
		return
	}
	if data.Password.IsNull() {
		password = os.Getenv("AXUAL_AUTH_PASSWORD")
		if password == "" {
			resp.Diagnostics.AddError(
				"Missing Password",
				"Password is not provided in configuration and the AXUAL_AUTH_PASSWORD environment variable is not set.",
			)
			return
		}
	} else {
		password = data.Password.ValueString()
	}

	auth := webclient.AuthStruct{
		Username: username,
		Password: password,
		Url:      data.AuthUrl.ValueString(),
		ClientId: data.ClientID.ValueString(),
	}

	if !data.Scopes.IsNull() {
		var scopes []string
		for _, s := range data.Scopes.Elements() {
			scopes = append(scopes, strings.Trim(s.String(), "\""))
		}
		auth.Scopes = scopes
	}

	c, err := webclient.NewClient(apiurl, realm, auth)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create Axual client:\n\n"+err.Error(),
		)
		return
	}

	p.client = c
}

func (p *AxualProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return NewApplicationResource(*p) },
		func() resource.Resource { return NewUserResource(*p) },
		func() resource.Resource { return NewGroupResource(*p) },
		func() resource.Resource { return NewTopicResource(*p) },
		func() resource.Resource { return NewTopicConfigResource(*p) },
		func() resource.Resource { return NewEnvironmentResource(*p) },
		func() resource.Resource { return NewApplicationPrincipalResource(*p) },
		func() resource.Resource { return NewSchemaVersionResource(*p) },
		func() resource.Resource { return NewApplicationAccessGrantResource(*p) },
		func() resource.Resource { return NewApplicationAccessGrantRejectionResource(*p) },
		func() resource.Resource { return NewApplicationAccessGrantApprovalResource(*p) },
		func() resource.Resource { return NewApplicationDeploymentResource(*p) },
		func() resource.Resource { return NewTopicBrowsePermissionsResource(*p) },
	}
}

func (p *AxualProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "axual"
	resp.Version = p.version
}

func (p *AxualProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func (p *AxualProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return NewApplicationDataSource(*p) },
		func() datasource.DataSource { return NewGroupDataSource(*p) },
		func() datasource.DataSource { return NewTopicDataSource(*p) },
		func() datasource.DataSource { return NewEnvironmentDataSource(*p) },
		func() datasource.DataSource { return NewSchemaVersionDataSource(*p) },
		func() datasource.DataSource { return NewApplicationAccessGrantDataSource(*p) },
		func() datasource.DataSource { return NewInstanceDataSource(*p) },
	}
}

func (p *AxualProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"apiurl": schema.StringAttribute{
				MarkdownDescription: "URL that will be used by the client for all resource requests",
				Required:            true,
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Axual realm used for the requests",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for all requests. Will be used to acquire a token",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password belonging to the user",
				Optional:            true,
				Sensitive:           true,
			},
			"clientid": schema.StringAttribute{
				MarkdownDescription: "Client ID to be used for oauth",
				Required:            true,
			},
			"authurl": schema.StringAttribute{
				MarkdownDescription: "Token url",
				Required:            true,
			},
			"scopes": schema.ListAttribute{
				MarkdownDescription: "OAuth authorization server scopes",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AxualProvider{
			version: version,
		}
	}
}

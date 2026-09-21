package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"

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
	ApiUrl types.String `tfsdk:"apiurl"`
	Realm  types.String `tfsdk:"realm"`

	ClientID      types.String `tfsdk:"client_id"`
	ClientSecret  types.String `tfsdk:"client_secret"`
	OIDCToken     types.String `tfsdk:"oidc_token"`
	OIDCTokenFile types.String `tfsdk:"oidc_token_file"`

	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	LegacyClientID types.String `tfsdk:"clientid"`
	AuthUrl        types.String `tfsdk:"authurl"`
	Scopes         types.List   `tfsdk:"scopes"`
	Audience       types.String `tfsdk:"audience"`
	AuthMode       types.String `tfsdk:"authmode"`
}

func (p *AxualProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A value that is still unknown at configure time cannot be used to reach the API or
	// sign in, and the resulting failure would point at the wrong thing.
	// A slice, not a map: with more than one unknown the reported attribute must be the
	// same on every run.
	configAttributes := []struct {
		name  string
		value types.String
	}{
		{"apiurl", data.ApiUrl},
		{"authurl", data.AuthUrl},
		{"realm", data.Realm},
		{"client_id", data.ClientID},
		{"client_secret", data.ClientSecret},
		{"oidc_token", data.OIDCToken},
		{"oidc_token_file", data.OIDCTokenFile},
		{"username", data.Username},
		{"password", data.Password},
		{"clientid", data.LegacyClientID},
	}
	for _, attribute := range configAttributes {
		if attribute.value.IsUnknown() {
			resp.Diagnostics.AddError(
				"Provider configuration is not known yet",
				fmt.Sprintf("The value of %q is not known until after apply, so the provider "+
					"cannot be configured with it. Supply it from a variable or an environment "+
					"variable instead of from another resource's output.", attribute.name),
			)
			return
		}
	}

	var scopes []string
	if !data.Scopes.IsNull() {
		resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	credentials, warnings, err := webclient.Resolve(webclient.ProviderConfig{
		AuthURL:        data.AuthUrl.ValueString(),
		ClientID:       data.ClientID.ValueString(),
		LegacyClientID: data.LegacyClientID.ValueString(),
		ClientSecret:   data.ClientSecret.ValueString(),
		OIDCToken:      data.OIDCToken.ValueString(),
		OIDCTokenFile:  data.OIDCTokenFile.ValueString(),
		Username:       data.Username.ValueString(),
		Password:       data.Password.ValueString(),
		AuthMode:       data.AuthMode.ValueString(),
		Scopes:         scopes,
	}, os.Getenv)

	for _, warning := range warnings {
		resp.Diagnostics.AddWarning(warning.Summary, warning.Detail)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to determine how to authenticate", err.Error())
		return
	}

	tflog.Info(ctx, "Resolved Axual provider authentication", map[string]interface{}{
		"mode":    string(credentials.Mode),
		"sources": credentials.Sources,
	})

	client, err := webclient.NewClient(data.ApiUrl.ValueString(), data.Realm.ValueString(), credentials)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create Axual client:\n\n"+err.Error(),
		)
		return
	}

	p.client = client
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
		func() resource.Resource { return NewApplicationCredentialResource(*p) },
		func() resource.Resource { return NewFlinkClusterResource(*p) },
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
		func() datasource.DataSource { return NewUserDataSource(*p) },
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
				MarkdownDescription: "Axual realm used for the requests. This is your tenant's short name.",
				Optional:            true,
			},
			"authurl": schema.StringAttribute{
				MarkdownDescription: "Token URL",
				Required:            true,
			},

			"client_id": schema.StringAttribute{
				MarkdownDescription: "Client ID of the service account. Not used by federated service accounts, " +
					"which Keycloak identifies from the assertion. Can be omitted if the environment " +
					"variable `AXUAL_CLIENT_ID` is set.",
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Client secret of the service account. Setting it selects service account " +
					"authentication with a secret. Can be omitted if the environment variable " +
					"`AXUAL_CLIENT_SECRET` is set.",
				Optional:  true,
				Sensitive: true,
			},
			"oidc_token": schema.StringAttribute{
				MarkdownDescription: "The assertion a federated service account authenticates with: a token " +
					"issued by your own identity provider, not an Axual token. Setting it selects " +
					"federated authentication, which sends no client secret at all. Can be omitted if " +
					"the environment variable `AXUAL_OIDC_TOKEN` is set, which is the usual way to " +
					"supply it from a CI pipeline.",
				Optional:  true,
				Sensitive: true,
			},
			"oidc_token_file": schema.StringAttribute{
				MarkdownDescription: "Path to a file holding the assertion a federated service account " +
					"authenticates with. The file is read again every time the access token is renewed, " +
					"so use this rather than `oidc_token` when something rewrites the assertion as it " +
					"rotates. Can be omitted if the environment variable `AXUAL_OIDC_TOKEN_FILE` is set.",
				Optional: true,
			},

			"username": schema.StringAttribute{
				MarkdownDescription: "Username for all requests. Will be used to acquire a token. " +
					"Deprecated: use a service account instead.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password belonging to the user. Deprecated: use a service account instead.",
				Optional:            true,
				Sensitive:           true,
			},
			"scopes": schema.ListAttribute{
				MarkdownDescription: "OAuth authorization server scopes. Only applies to username and password " +
					"authentication; service account authentication does not send a scope.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"clientid": schema.StringAttribute{
				MarkdownDescription: "Client ID to be used for oauth.",
				Optional:            true,
				DeprecationMessage:  "Use client_id instead.",
			},
			"audience": schema.StringAttribute{
				MarkdownDescription: "Audience for OAUTH.",
				Optional:            true,
				DeprecationMessage: "No longer has any effect. Will be removed in a future major release; " +
					"remove it from your configuration.",
			},
			"authmode": schema.StringAttribute{
				MarkdownDescription: "Authentication mode.",
				Optional:            true,
				DeprecationMessage: "No longer has any effect. The authentication method is determined by " +
					"which credentials are supplied. Remove it from your configuration.",
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

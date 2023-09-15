package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.Provider = &provider{}

// provider satisfies the tfsdk.Provider interface and usually is included
// with all Resource and DataSource implementations.
type provider struct {
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

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiurl := data.ApiUrl.Value
	realm := data.Realm.Value

	var username string
	if data.Username.Null {
		username = os.Getenv("AXUAL_AUTH_USERNAME")
		if username == "" {
			resp.Diagnostics.AddError(
				"Missing Username",
				"Username is not provided in configuration and the AXUAL_AUTH_USERNAME environment variable is not set.",
			)
			return
		}
	} else {
		username = data.Username.Value
	}

	var password string
	if data.Password.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as host",
		)
		return
	}
	if data.Password.Null {
		password = os.Getenv("AXUAL_AUTH_PASSWORD")
		if password == "" {
			resp.Diagnostics.AddError(
				"Missing Password",
				"Password is not provided in configuration and the AXUAL_AUTH_PASSWORD environment variable is not set.",
			)
			return
		}
	} else {
		password = data.Password.Value
	}

	auth := webclient.AuthStruct{
		Username: username,
		Password: password,
		Url:      data.AuthUrl.Value,
		ClientId: data.ClientID.Value,
	}

	if !data.Scopes.Null {
		var scopes []string
		for _, s := range data.Scopes.Elems {
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

func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"axual_user":                               userResourceType{},
		"axual_group":                              groupResourceType{},
		"axual_environment":                        environmentResourceType{},
		"axual_topic":                              topicResourceType{},
		"axual_topic_config":                       topicConfigResourceType{},
		"axual_application":                        applicationResourceType{},
		"axual_application_principal":              applicationPrincipalResourceType{},
		"axual_application_access_grant":           applicationAccessGrantResourceType{},
		"axual_application_access_grant_approval":  applicationAccessGrantApprovalResourceType{},
		"axual_application_access_grant_rejection": applicationAccessGrantRejectionResourceType{},
		"axual_schema_version":                     schemaVersionResourceType{},
	}, nil
}

func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"axual_environment":   environmentDataSourceType{},
	}, nil
}

func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"apiurl": {
				MarkdownDescription: "URL that will be used by the client for all resource requests",
				Required:            true,
				Type:                types.StringType,
			},
			"realm": {
				MarkdownDescription: "Axual realm used for the requests",
				Required:            true,
				Type:                types.StringType,
			},
			"username": {
				MarkdownDescription: "Username for all requests. Will be used to acquire a token",
				Optional:            true,
				Type:                types.StringType,
			},
			"password": {
				MarkdownDescription: "Password belonging to the user",
				Optional:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
			"clientid": {
				MarkdownDescription: "Client ID to be used for oauth",
				Required:            true,
				Type:                types.StringType,
			},
			"authurl": {
				MarkdownDescription: "Token url",
				Required:            true,
				Type:                types.StringType,
			},
			"scopes": {
				MarkdownDescription: "OAuth authorization server scopes",
				Optional:            true,
				Type:                types.ListType{ElemType: types.StringType},
			},
		},
	}, nil
}

func New(version string) func() tfsdk.Provider {
	return func() tfsdk.Provider {
		return &provider{
			version: version,
		}
	}
}

// convertProviderType is a helper function for NewResource and NewDataSource
// implementations to associate the concrete provider type. Alternatively,
// this helper can be skipped and the provider type can be directly type
// asserted (e.g. provider: in.(*provider)), however using this can prevent
// potential panics.
func convertProviderType(in tfsdk.Provider) (provider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*provider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return provider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return provider{}, diags
	}

	return *p, diags
}

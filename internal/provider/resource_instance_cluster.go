package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &instanceClusterResource{}
var _ resource.ResourceWithImportState = &instanceClusterResource{}

func NewInstanceClusterResource(provider AxualProvider) resource.Resource {
	return &instanceClusterResource{
		provider: provider,
	}
}

type instanceClusterResource struct {
	provider AxualProvider
}

type instanceClusterResourceData struct {
	Id             types.String `tfsdk:"id"`
	Instance       types.String `tfsdk:"instance"`
	Cluster        types.String `tfsdk:"cluster"`
	SchemaRegistry types.Object `tfsdk:"schema_registry"`
}

type schemaRegistryData struct {
	Type           types.String `tfsdk:"type"`
	Url            types.String `tfsdk:"url"`
	Version        types.String `tfsdk:"version"`
	Authentication types.String `tfsdk:"authentication"`
	Username       types.String `tfsdk:"username"`
	Password       types.String `tfsdk:"password"`
	Certificate    types.String `tfsdk:"certificate"`
	PrivateKey     types.String `tfsdk:"private_key"`
	Truststore     types.String `tfsdk:"truststore"`
	OauthEnabled   types.Bool   `tfsdk:"oauth_enabled"`
	Keycloak       types.Object `tfsdk:"keycloak"`
	Listeners      types.Map    `tfsdk:"listeners"`
	Properties     types.Map    `tfsdk:"properties"`
}

type schemaRegistryKeycloakData struct {
	Url           types.String `tfsdk:"url"`
	MasterRealm   types.String `tfsdk:"master_realm"`
	AdminUsername types.String `tfsdk:"admin_username"`
	AdminPassword types.String `tfsdk:"admin_password"`
	AdminClientId types.String `tfsdk:"admin_client_id"`
	ApicurioRealm types.String `tfsdk:"apicurio_realm"`
}

func schemaRegistryKeycloakAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"url":             types.StringType,
		"master_realm":    types.StringType,
		"admin_username":  types.StringType,
		"admin_password":  types.StringType,
		"admin_client_id": types.StringType,
		"apicurio_realm":  types.StringType,
	}
}

func schemaRegistryAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":           types.StringType,
		"url":            types.StringType,
		"version":        types.StringType,
		"authentication": types.StringType,
		"username":       types.StringType,
		"password":       types.StringType,
		"certificate":    types.StringType,
		"private_key":    types.StringType,
		"truststore":     types.StringType,
		"oauth_enabled":  types.BoolType,
		"keycloak":       types.ObjectType{AttrTypes: schemaRegistryKeycloakAttrTypes()},
		"listeners":      types.MapType{ElemType: types.StringType},
		"properties":     types.MapType{ElemType: types.StringType},
	}
}

func (r *instanceClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_cluster"
}

func (r *instanceClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The configuration of one Cluster in one Instance: its schema registry. The Cluster must be in the Instance's `clusters`. Platform Manager stores the credentials in Vault and does not contact the registry to check them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Instance ID and the Cluster ID, as `<instance>/<cluster>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance": schema.StringAttribute{
				MarkdownDescription: "The ID of the Instance. Changing it creates a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster": schema.StringAttribute{
				MarkdownDescription: "The ID of the Cluster. Changing it creates a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema_registry": schema.SingleNestedAttribute{
				MarkdownDescription: "The schema registry topics with schemas on this Cluster use.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "`APICURIO` or `CONFLUENT`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("APICURIO", "CONFLUENT"),
						},
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "Base URL of the registry API Platform Manager uses, for example `https://apicurio.example.com/apis/registry/v3`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"version": schema.StringAttribute{
						MarkdownDescription: "The registry version, for information.",
						Optional:            true,
					},
					"authentication": schema.StringAttribute{
						MarkdownDescription: "How Platform Manager signs in to the registry: `Basic-Auth`, `MTLS` or `No-Auth`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("Basic-Auth", "MTLS", "No-Auth"),
						},
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "Username for `Basic-Auth`. Requires `password`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("password")),
						},
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "Password for `Basic-Auth`. Platform Manager never returns it, so it is not read back and cannot be imported. This field is Sensitive.",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("username")),
						},
					},
					"certificate": schema.StringAttribute{
						MarkdownDescription: "PEM client certificate for `MTLS`. Requires `private_key`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("private_key")),
						},
					},
					"private_key": schema.StringAttribute{
						MarkdownDescription: "PEM private key of `certificate`. Platform Manager never returns it. This field is Sensitive.",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("certificate")),
						},
					},
					"truststore": schema.StringAttribute{
						MarkdownDescription: "PEM CA certificates that sign the registry's certificate.",
						Optional:            true,
					},
					"oauth_enabled": schema.BoolAttribute{
						MarkdownDescription: "`APICURIO` only: whether applications may also sign in with OAuth. Needs `keycloak`. Basic authentication is always enabled. Default `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"keycloak": schema.SingleNestedAttribute{
						MarkdownDescription: "`APICURIO` only: the Keycloak behind the registry, where Platform Manager creates the clients of applications that use OAuth.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"url":             schema.StringAttribute{Required: true, MarkdownDescription: "Keycloak base URL."},
							"master_realm":    schema.StringAttribute{Required: true, MarkdownDescription: "The realm the admin user signs in to, usually `master`."},
							"admin_username":  schema.StringAttribute{Required: true, MarkdownDescription: "Keycloak admin username."},
							"admin_password":  schema.StringAttribute{Required: true, Sensitive: true, MarkdownDescription: "Keycloak admin password. Platform Manager never returns it. This field is Sensitive."},
							"admin_client_id": schema.StringAttribute{Required: true, MarkdownDescription: "The client the admin user signs in with, usually `admin-cli`."},
							"apicurio_realm":  schema.StringAttribute{Required: true, MarkdownDescription: "The realm of the Apicurio clients."},
						},
					},
					"listeners": schema.MapAttribute{
						MarkdownDescription: "Registry URLs applications use, per authentication method.",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
					"properties": schema.MapAttribute{
						MarkdownDescription: "Extra settings Platform Manager passes to the registry client.",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *instanceClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data instanceClusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &data, &resp.Diagnostics, "CREATE request error for instance cluster resource")
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "configured Instance-Cluster schema registry")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data instanceClusterResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, err := r.provider.client.GetInstanceClusterSchemaRegistry(data.Instance.ValueString(), data.Cluster.ValueString())
	if err != nil && !isGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Instance-Cluster schema registry, got error: %s", err))
		return
	}
	// Platform Manager answers an empty registry when none is configured.
	if err != nil || registry.Url == "" {
		tflog.Warn(ctx, fmt.Sprintf("Instance-Cluster schema registry not found. Id: %s", data.Id.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(mapSchemaRegistryResponseToData(ctx, &data, registry)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data instanceClusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &data, &resp.Diagnostics, "Unable to update Instance-Cluster schema registry")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// write posts the whole schema registry (the POST creates or replaces it) and reads it back.
func (r *instanceClusterResource) write(ctx context.Context, data *instanceClusterResourceData, diags *diag.Diagnostics, errorSummary string) {
	registry, d := createSchemaRegistryRequestFromData(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	instance, cluster := data.Instance.ValueString(), data.Cluster.ValueString()
	if err := r.provider.client.SetInstanceClusterSchemaRegistry(instance, cluster, registry); err != nil {
		diags.AddError(errorSummary, fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	answer, err := r.provider.client.GetInstanceClusterSchemaRegistry(instance, cluster)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read Instance-Cluster schema registry, got error: %s", err))
		return
	}
	data.Id = types.StringValue(instance + "/" + cluster)
	diags.Append(mapSchemaRegistryResponseToData(ctx, data, answer)...)
}

func (r *instanceClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data instanceClusterResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := ignoreGone(r.provider.client.DeleteInstanceClusterSchemaRegistry(data.Instance.ValueString(), data.Cluster.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Instance-Cluster schema registry, got error: %s", err))
	}
}

// ImportState accepts "<instance id>/<cluster id>", which is also the resource's id.
func (r *instanceClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id/cluster_id. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster"), parts[1])...)
}

func createSchemaRegistryRequestFromData(ctx context.Context, data *instanceClusterResourceData) (webclient.InstanceClusterSchemaRegistry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var registry schemaRegistryData
	diags.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)

	request := webclient.InstanceClusterSchemaRegistry{
		Url:                   registry.Url.ValueString(),
		SchemaRegistryType:    registry.Type.ValueString(),
		SchemaRegistryVersion: stringPointerOrNil(registry.Version),
		Credentials: &webclient.SchemaRegistryCredentials{
			AuthenticationType: registry.Authentication.ValueString(),
			Username:           registry.Username.ValueString(),
			Password:           registry.Password.ValueString(),
			Certificate:        pemToAPI(registry.Certificate.ValueString()),
			PrivateKey:         pemToAPI(registry.PrivateKey.ValueString()),
			Truststore:         pemToAPI(registry.Truststore.ValueString()),
		},
		Listeners:  stringMapFromData(registry.Listeners),
		Properties: stringMapFromData(registry.Properties),
	}
	if registry.Type.ValueString() == "APICURIO" {
		request.Authentications = &webclient.SchemaRegistryAuthentications{Basic: true, OAuth: registry.OauthEnabled.ValueBool()}
	}
	if !registry.Keycloak.IsNull() && !registry.Keycloak.IsUnknown() {
		var keycloak schemaRegistryKeycloakData
		diags.Append(registry.Keycloak.As(ctx, &keycloak, basetypes.ObjectAsOptions{})...)
		request.KeycloakDetails = &webclient.SchemaRegistryKeycloakDetails{
			Url:           keycloak.Url.ValueString(),
			MasterRealm:   keycloak.MasterRealm.ValueString(),
			AdminUsername: keycloak.AdminUsername.ValueString(),
			AdminPassword: keycloak.AdminPassword.ValueString(),
			AdminClientId: keycloak.AdminClientId.ValueString(),
			ApicurioRealm: keycloak.ApicurioRealm.ValueString(),
		}
	}
	return request, diags
}

// mapSchemaRegistryResponseToData writes the API's view into state. The password, private key and
// Keycloak admin password are kept as they were: Platform Manager answers them as "".
func mapSchemaRegistryResponseToData(ctx context.Context, data *instanceClusterResourceData, answer *webclient.InstanceClusterSchemaRegistry) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Id = types.StringValue(data.Instance.ValueString() + "/" + data.Cluster.ValueString())

	var registry schemaRegistryData
	if !data.SchemaRegistry.IsNull() && !data.SchemaRegistry.IsUnknown() {
		diags.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
	}
	registry.Type = types.StringValue(strings.ToUpper(answer.SchemaRegistryType))
	registry.Url = types.StringValue(answer.Url)
	if answer.SchemaRegistryVersion != nil {
		registry.Version = stringOrNull(*answer.SchemaRegistryVersion)
	} else {
		registry.Version = types.StringNull()
	}
	credentials := webclient.SchemaRegistryCredentials{}
	if answer.Credentials != nil {
		credentials = *answer.Credentials
	}
	registry.Authentication = stringOrNull(credentials.AuthenticationType)
	registry.Username = stringOrNull(credentials.Username)
	registry.Certificate = stringOrNull(pemFromAPI(credentials.Certificate, registry.Certificate))
	registry.Truststore = stringOrNull(pemFromAPI(credentials.Truststore, registry.Truststore))
	if registry.Password.IsUnknown() {
		registry.Password = types.StringNull()
	}
	if registry.PrivateKey.IsUnknown() {
		registry.PrivateKey = types.StringNull()
	}
	registry.OauthEnabled = types.BoolValue(answer.Authentications != nil && answer.Authentications.OAuth)
	registry.Listeners = stringMapToData(answer.Listeners)
	registry.Properties = stringMapToData(answer.Properties)

	if answer.KeycloakDetails == nil || answer.KeycloakDetails.Url == "" {
		registry.Keycloak = types.ObjectNull(schemaRegistryKeycloakAttrTypes())
	} else {
		var keycloak schemaRegistryKeycloakData
		if !registry.Keycloak.IsNull() && !registry.Keycloak.IsUnknown() {
			diags.Append(registry.Keycloak.As(ctx, &keycloak, basetypes.ObjectAsOptions{})...)
		}
		keycloak.Url = types.StringValue(answer.KeycloakDetails.Url)
		keycloak.MasterRealm = types.StringValue(answer.KeycloakDetails.MasterRealm)
		keycloak.AdminUsername = types.StringValue(answer.KeycloakDetails.AdminUsername)
		keycloak.AdminClientId = types.StringValue(answer.KeycloakDetails.AdminClientId)
		keycloak.ApicurioRealm = types.StringValue(answer.KeycloakDetails.ApicurioRealm)
		if keycloak.AdminPassword.IsUnknown() || keycloak.AdminPassword.IsNull() {
			keycloak.AdminPassword = types.StringValue("")
		}
		object, d := types.ObjectValueFrom(ctx, schemaRegistryKeycloakAttrTypes(), keycloak)
		diags.Append(d...)
		registry.Keycloak = object
	}

	object, d := types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
	diags.Append(d...)
	data.SchemaRegistry = object
	return diags
}

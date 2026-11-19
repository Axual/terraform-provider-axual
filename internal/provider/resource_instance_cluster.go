package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

// Platform Manager knows the api and the client endpoints of a schema registry from 16.1.0 on. Older
// versions ignore them in a request and do not return them.
const schemaRegistryApiMajor, schemaRegistryApiMinor = 16, 1

// schemaRegistryTypeOfApi is the type each schema registry API belongs to.
var schemaRegistryTypeOfApi = map[string]string{"APICURIO_V2": "APICURIO", "APICURIO_V3": "APICURIO", "CONFLUENT_V7": "CONFLUENT"}

var schemaRegistryApis = []string{"APICURIO_V2", "APICURIO_V3", "CONFLUENT_V7"}

var schemaRegistryAuthentications = []string{"Basic-Auth", "MTLS", "No-Auth"}

// Platform Manager 16.1.0 saves the whole Instance on every schema registry change, so parallel changes on
// one Instance can lose its optimistic lock (403). They are retried. Variables so the unit tests can shorten them.
var (
	schemaRegistryWriteAttempts = 10
	schemaRegistryWriteDelay    = time.Second
)

var _ resource.Resource = &instanceClusterResource{}
var _ resource.ResourceWithImportState = &instanceClusterResource{}
var _ resource.ResourceWithModifyPlan = &instanceClusterResource{}
var _ resource.ResourceWithValidateConfig = &instanceClusterResource{}

// SupportsSchemaRegistryApi reports whether a Platform Manager version, as GET /info returns it, has the
// api and the client endpoints of a schema registry. A version it cannot read counts as too old.
func SupportsSchemaRegistryApi(version string) bool {
	return platformVersionAtLeast(version, schemaRegistryApiMajor, schemaRegistryApiMinor)
}

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
	Type            types.String `tfsdk:"type"`
	Url             types.String `tfsdk:"url"`
	Version         types.String `tfsdk:"version"`
	Authentication  types.String `tfsdk:"authentication"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	Certificate     types.String `tfsdk:"certificate"`
	PrivateKey      types.String `tfsdk:"private_key"`
	Truststore      types.String `tfsdk:"truststore"`
	OauthEnabled    types.Bool   `tfsdk:"oauth_enabled"`
	Keycloak        types.Object `tfsdk:"keycloak"`
	Listeners       types.Map    `tfsdk:"listeners"`
	Properties      types.Map    `tfsdk:"properties"`
	Api             types.String `tfsdk:"api"`
	ClientEndpoints types.Map    `tfsdk:"client_endpoints"`
}

type schemaRegistryClientEndpointData struct {
	Api    types.String `tfsdk:"api"`
	Url    types.String `tfsdk:"url"`
	UsedBy types.String `tfsdk:"used_by"`
}

func schemaRegistryClientEndpointType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"api":     types.StringType,
		"url":     types.StringType,
		"used_by": types.StringType,
	}}
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
		"type":             types.StringType,
		"url":              types.StringType,
		"version":          types.StringType,
		"authentication":   types.StringType,
		"username":         types.StringType,
		"password":         types.StringType,
		"certificate":      types.StringType,
		"private_key":      types.StringType,
		"truststore":       types.StringType,
		"oauth_enabled":    types.BoolType,
		"keycloak":         types.ObjectType{AttrTypes: schemaRegistryKeycloakAttrTypes()},
		"listeners":        types.MapType{ElemType: types.StringType},
		"properties":       types.MapType{ElemType: types.StringType},
		"api":              types.StringType,
		"client_endpoints": types.MapType{ElemType: schemaRegistryClientEndpointType()},
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
						MarkdownDescription: "The registry version, for example `3.0`. From Platform Manager 16.1.0, it also sets the API of an `APICURIO` registry: a version that starts with `2` means the Apicurio 2.x API, any other version the 3.x API.",
						Optional:            true,
						Validators:          []validator.String{notEmpty},
					},
					"authentication": schema.StringAttribute{
						MarkdownDescription: "How Platform Manager signs in to the registry: `Basic-Auth`, `MTLS` or `No-Auth`. `MTLS` is stored as `mTLS`, the spelling of the Self-Service UI.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(schemaRegistryAuthentications...),
						},
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "Username for `Basic-Auth`. Requires `password`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("password")),
							notEmpty,
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
							notEmpty,
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
						Validators:          []validator.String{notEmpty},
					},
					"oauth_enabled": schema.BoolAttribute{
						MarkdownDescription: "`APICURIO` only: whether applications may also sign in with OAuth. Basic authentication is always enabled. Like a Basic-Auth credential, an OAuth credential needs `keycloak` on this or another Cluster of the Instance. An application gets its OAuth credential in the Self-Service UI: `axual_application_credential` makes Basic-Auth credentials only. Default `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"keycloak": schema.SingleNestedAttribute{
						MarkdownDescription: "`APICURIO` only, and not with `No-Auth`: the Keycloak behind the registry. Platform Manager creates the client of every schema registry credential of an application there, Basic-Auth and OAuth, so at least one Cluster of the Instance needs it. Leaving it out removes the stored Keycloak details and admin password.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"url":             schema.StringAttribute{Required: true, Validators: []validator.String{notEmpty}, MarkdownDescription: "Keycloak base URL."},
							"master_realm":    schema.StringAttribute{Required: true, MarkdownDescription: "The realm the admin user signs in to, usually `master`."},
							"admin_username":  schema.StringAttribute{Required: true, MarkdownDescription: "Keycloak admin username."},
							"admin_password":  schema.StringAttribute{Required: true, Sensitive: true, MarkdownDescription: "Keycloak admin password. Platform Manager never returns it, and signs in to Keycloak with it on every save of the registry, so a wrong one fails the apply. This field is Sensitive."},
							"admin_client_id": schema.StringAttribute{Required: true, MarkdownDescription: "The client the admin user signs in with, usually `admin-cli`."},
							"apicurio_realm":  schema.StringAttribute{Required: true, MarkdownDescription: "The realm of the Apicurio clients."},
						},
					},
					"listeners": schema.MapAttribute{
						MarkdownDescription: "Registry URLs applications use, by name. From Platform Manager 16.1.0, these are the `client_endpoints`: a new name becomes an endpoint with the registry's `api` for `ALL_APPS`, and an endpoint whose name is left out is removed. Conflicts with `client_endpoints`.",
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
					"api": schema.StringAttribute{
						MarkdownDescription: "The API the registry at `url` speaks: `APICURIO_V2` or `APICURIO_V3` for `APICURIO`, `CONFLUENT_V7` for `CONFLUENT`. Needs Platform Manager 16.1.0 or later. When it is not set, Platform Manager derives it from `type` and `version`, and the provider keeps it until `type` or `version` changes.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(schemaRegistryApis...),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"client_endpoints": schema.MapNestedAttribute{
						MarkdownDescription: "The registry endpoints applications use, by name. Needs Platform Manager 16.1.0 or later. When set, it replaces all endpoints of the Instance-Cluster, and an endpoint keeps its identity in Platform Manager while its name stays the same. Leaving it out keeps the endpoints Platform Manager has, and `{}` removes them all. When it is not set, the endpoints follow `listeners`. Conflicts with `listeners`, which Platform Manager ignores when the endpoints are sent.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.Map{
							mapvalidator.KeysAre(stringvalidator.LengthBetween(1, 255)),
							mapvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("listeners")),
						},
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"api": schema.StringAttribute{
									MarkdownDescription: "The API applications use at this endpoint: `APICURIO_V2`, `APICURIO_V3` or `CONFLUENT_V7`.",
									Required:            true,
									Validators:          []validator.String{stringvalidator.OneOf(schemaRegistryApis...)},
								},
								"url": schema.StringAttribute{
									MarkdownDescription: "The URL of the endpoint, at most 255 characters.",
									Required:            true,
									Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
								},
								"used_by": schema.StringAttribute{
									MarkdownDescription: "Which applications use the endpoint: `MANAGED_APPS`, or `ALL_APPS` (managed and self-managed applications; only these endpoints are shown to users).",
									Required:            true,
									Validators:          []validator.String{stringvalidator.OneOf("MANAGED_APPS", "ALL_APPS")},
								},
							},
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

// ValidateConfig refuses OAuth on a registry that is not APICURIO, which Platform Manager reads back as false,
// and an api of another type than type, which Platform Manager refuses.
func (r *instanceClusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data instanceClusterResourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.SchemaRegistry.IsNull() || data.SchemaRegistry.IsUnknown() {
		return
	}
	var registry schemaRegistryData
	resp.Diagnostics.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || registry.Type.IsUnknown() {
		return
	}
	if registry.OauthEnabled.ValueBool() && registry.Type.ValueString() != "APICURIO" {
		resp.Diagnostics.AddAttributeError(path.Root("schema_registry").AtName("oauth_enabled"), "oauth_enabled needs type APICURIO",
			"Platform Manager supports OAuth only for an APICURIO schema registry, and reads it back as false for "+registry.Type.ValueString()+".")
	}
	if !registry.Keycloak.IsNull() && !registry.Keycloak.IsUnknown() &&
		(registry.Type.ValueString() != "APICURIO" || registry.Authentication.ValueString() == "No-Auth") {
		resp.Diagnostics.AddAttributeError(path.Root("schema_registry").AtName("keycloak"), "keycloak needs type APICURIO with authentication",
			`Platform Manager refuses keycloak for a CONFLUENT registry and with authentication "No-Auth": "Keycloak details can only be provided for APICURIO schema registry with enabled authentication".`)
	}
	if registry.Api.IsNull() || registry.Api.IsUnknown() {
		return
	}
	if want := schemaRegistryTypeOfApi[registry.Api.ValueString()]; want != "" && want != registry.Type.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("schema_registry").AtName("api"), "api does not match type",
			fmt.Sprintf("api %s is for a %s registry, but type is %s.", registry.Api.ValueString(), want, registry.Type.ValueString()))
	}
}

// ModifyPlan plans api, client_endpoints and listeners as unknown when they are not set and Platform
// Manager derives them again from a change, and refuses api and client_endpoints on a Platform Manager
// before 16.1.0, which ignores them.
func (r *instanceClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config, plan instanceClusterResourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || config.SchemaRegistry.IsUnknown() || plan.SchemaRegistry.IsUnknown() {
		return
	}
	var configured, planned schemaRegistryData
	resp.Diagnostics.Append(config.SchemaRegistry.As(ctx, &configured, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(plan.SchemaRegistry.As(ctx, &planned, basetypes.ObjectAsOptions{})...)
	var current *schemaRegistryData
	if !req.State.Raw.IsNull() {
		var state instanceClusterResourceData
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		current = &schemaRegistryData{}
		resp.Diagnostics.Append(state.SchemaRegistry.As(ctx, current, basetypes.ObjectAsOptions{})...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if current != nil {
		changed := false
		if configured.Api.IsNull() && (!planned.Type.Equal(current.Type) || !planned.Version.Equal(current.Version)) {
			planned.Api, changed = types.StringUnknown(), true
		}
		if configured.ClientEndpoints.IsNull() && !planned.Listeners.Equal(current.Listeners) {
			planned.ClientEndpoints, changed = types.MapUnknown(schemaRegistryClientEndpointType()), true
		}
		if configured.Listeners.IsNull() && !planned.ClientEndpoints.Equal(current.ClientEndpoints) {
			planned.Listeners, changed = types.MapUnknown(types.StringType), true
		}
		if changed {
			object, d := types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), planned)
			resp.Diagnostics.Append(d...)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("schema_registry"), object)...)
		}
	}

	newApi := !configured.Api.IsNull() && (current == nil || !planned.Api.Equal(current.Api))
	newEndpoints := !configured.ClientEndpoints.IsNull() && (current == nil || !planned.ClientEndpoints.Equal(current.ClientEndpoints))
	if (!newApi && !newEndpoints) || r.provider.client == nil {
		return
	}
	version, err := r.provider.client.GetPlatformVersion()
	if newApi {
		requirePlatformVersion(&resp.Diagnostics, version, err, path.Root("schema_registry").AtName("api"), "api",
			schemaRegistryApiMajor, schemaRegistryApiMinor, "It ignores api, so the apply would fail.")
	}
	if newEndpoints {
		requirePlatformVersion(&resp.Diagnostics, version, err, path.Root("schema_registry").AtName("client_endpoints"), "client_endpoints",
			schemaRegistryApiMajor, schemaRegistryApiMinor, "It ignores client_endpoints, so the apply would fail.")
	}
}

func (r *instanceClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config instanceClusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The POST creates or replaces, so a create must not overwrite a schema registry made elsewhere.
	instance, cluster := data.Instance.ValueString(), data.Cluster.ValueString()
	existing, err := r.provider.client.GetInstanceClusterSchemaRegistry(instance, cluster)
	if err != nil && !isGone(err) && !r.clusterLeftInstance(instance, cluster) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Instance-Cluster schema registry, got error: %s", err))
		return
	}
	if err == nil && existing.Url != "" {
		if !holdsPlannedValues(ctx, &data, existing) {
			resp.Diagnostics.AddError("Schema registry already configured",
				fmt.Sprintf("Instance-Cluster %s/%s already has a schema registry (%s). Import it to manage it with Terraform: terraform import <address> %s/%s",
					instance, cluster, existing.Url, instance, cluster))
			return
		}
		resp.Diagnostics.AddWarning("Schema registry taken over",
			fmt.Sprintf("Instance-Cluster %s/%s already had a schema registry with every setting of this configuration, for example from an earlier apply whose answer was lost. Terraform now manages it.",
				instance, cluster))
	}
	r.write(ctx, &data, &config, &resp.Diagnostics, "CREATE request error for instance cluster resource")
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
	if err != nil && !isGone(err) && !r.clusterLeftInstance(data.Instance.ValueString(), data.Cluster.ValueString()) {
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
	var data, config instanceClusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &data, &config, &resp.Diagnostics, "Unable to update Instance-Cluster schema registry")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// write posts the whole schema registry (the POST creates or replaces it) and reads it back. The
// client endpoints keep their ids: an endpoint with the name of an existing one is sent with its id,
// so Platform Manager updates it in place.
func (r *instanceClusterResource) write(ctx context.Context, data, config *instanceClusterResourceData, diags *diag.Diagnostics, errorSummary string) {
	registry, d := createSchemaRegistryRequestFromData(ctx, data, config)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	instance, cluster := data.Instance.ValueString(), data.Cluster.ValueString()
	err := RetryIfJittered(ctx, schemaRegistryWriteAttempts, schemaRegistryWriteDelay, isConflict, func() error {
		if registry.ClientEndpoints != nil {
			existing, err := r.provider.client.GetInstanceClusterSchemaRegistry(instance, cluster)
			if err != nil && !isGone(err) {
				return fmt.Errorf("reading the client endpoints: %w", err)
			}
			if err == nil {
				setClientEndpointIds(*registry.ClientEndpoints, existing)
			}
		}
		return r.provider.client.SetInstanceClusterSchemaRegistry(instance, cluster, registry)
	})
	if errors.Is(err, webclient.NotFoundError) {
		diags.AddError(errorSummary, fmt.Sprintf("Cluster %s is not in the clusters of Instance %s. Add it to the Instance first.", cluster, instance))
		return
	}
	if err != nil {
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

	err := ignoreGone(RetryIfJittered(ctx, schemaRegistryWriteAttempts, schemaRegistryWriteDelay, isConflict, func() error {
		return r.provider.client.DeleteInstanceClusterSchemaRegistry(data.Instance.ValueString(), data.Cluster.ValueString())
	}))
	if err != nil && !r.clusterLeftInstance(data.Instance.ValueString(), data.Cluster.ValueString()) {
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

// createSchemaRegistryRequestFromData builds the POST body from the plan. api is sent when it is known:
// set in the configuration, or kept from the state. client_endpoints is sent only when the
// configuration sets it, and listeners otherwise, so an older Platform Manager gets nothing new.
func createSchemaRegistryRequestFromData(ctx context.Context, data, config *instanceClusterResourceData) (webclient.InstanceClusterSchemaRegistry, diag.Diagnostics) {
	var diags diag.Diagnostics
	var registry, configured schemaRegistryData
	diags.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
	diags.Append(config.SchemaRegistry.As(ctx, &configured, basetypes.ObjectAsOptions{})...)

	request := webclient.InstanceClusterSchemaRegistry{
		Url:                   registry.Url.ValueString(),
		SchemaRegistryType:    registry.Type.ValueString(),
		SchemaRegistryVersion: stringPointerOrNil(registry.Version),
		Credentials: &webclient.SchemaRegistryCredentials{
			AuthenticationType: mtlsToAPI(registry.Authentication.ValueString()),
			Username:           registry.Username.ValueString(),
			Password:           nonEmptyPointer(registry.Password.ValueString()),
			Certificate:        pemToAPI(registry.Certificate.ValueString()),
			PrivateKey:         nonEmptyPointer(pemToAPI(registry.PrivateKey.ValueString())),
			Truststore:         pemToAPI(registry.Truststore.ValueString()),
		},
		Api:        stringPointerOrNil(registry.Api),
		Properties: stringMapFromData(registry.Properties),
	}
	if configured.ClientEndpoints.IsNull() {
		request.Listeners = stringMapFromData(registry.Listeners)
	} else {
		endpoints, d := clientEndpointsFromData(ctx, registry.ClientEndpoints)
		diags.Append(d...)
		request.ClientEndpoints = &endpoints
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

// clientEndpointsFromData returns the endpoints of the plan, sorted by name.
func clientEndpointsFromData(ctx context.Context, value types.Map) ([]webclient.SchemaRegistryClientEndpoint, diag.Diagnostics) {
	var diags diag.Diagnostics
	endpoints := []webclient.SchemaRegistryClientEndpoint{}
	var byName map[string]schemaRegistryClientEndpointData
	diags.Append(value.ElementsAs(ctx, &byName, false)...)
	for name, endpoint := range byName {
		endpoints = append(endpoints, webclient.SchemaRegistryClientEndpoint{
			Name: name, Api: endpoint.Api.ValueString(), Url: endpoint.Url.ValueString(), UsedBy: endpoint.UsedBy.ValueString(),
		})
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })
	return endpoints, diags
}

// setClientEndpointIds gives each endpoint the id of the existing endpoint with the same name.
func setClientEndpointIds(endpoints []webclient.SchemaRegistryClientEndpoint, existing *webclient.InstanceClusterSchemaRegistry) {
	if existing == nil || existing.ClientEndpoints == nil {
		return
	}
	ids := map[string]string{}
	for _, endpoint := range *existing.ClientEndpoints {
		ids[endpoint.Name] = endpoint.Id
	}
	for i := range endpoints {
		endpoints[i].Id = ids[endpoints[i].Name]
	}
}

// clientEndpointsToData maps the answer's endpoints to state: null when Platform Manager does not
// return them (before 16.1.0).
func clientEndpointsToData(endpoints *[]webclient.SchemaRegistryClientEndpoint) types.Map {
	if endpoints == nil {
		return types.MapNull(schemaRegistryClientEndpointType())
	}
	elements := map[string]attr.Value{}
	for _, endpoint := range *endpoints {
		elements[endpoint.Name] = types.ObjectValueMust(schemaRegistryClientEndpointType().AttrTypes, map[string]attr.Value{
			"api":     types.StringValue(endpoint.Api),
			"url":     types.StringValue(endpoint.Url),
			"used_by": types.StringValue(endpoint.UsedBy),
		})
	}
	return types.MapValueMust(schemaRegistryClientEndpointType(), elements)
}

// clusterLeftInstance reports whether the Instance exists and no longer has the Cluster, so the schema
// registry is gone too. Platform Manager 16.1.0 answers the registry requests of such a pair with a 403
// (LazyInitializationException) instead of a 404. It is asked only after such an error.
func (r *instanceClusterResource) clusterLeftInstance(instanceId, clusterId string) bool {
	instance, err := r.provider.client.GetInstance(instanceId)
	if err != nil {
		return false
	}
	for _, instanceCluster := range instance.InstanceClusters {
		if instanceCluster.Cluster.Uid == clusterId {
			return false
		}
	}
	return true
}

// lostSecrets sets the password and private key that Platform Manager no longer has to null, so the
// plan sends them again, and names them. Platform Manager answers a stored one as "".
func lostSecrets(registry *schemaRegistryData, credentials webclient.SchemaRegistryCredentials) string {
	var lost []string
	if registry.Password.ValueString() != "" && credentials.Password == nil {
		registry.Password = types.StringNull()
		lost = append(lost, "password")
	}
	if registry.PrivateKey.ValueString() != "" && credentials.PrivateKey == nil {
		registry.PrivateKey = types.StringNull()
		lost = append(lost, "private key")
	}
	return strings.Join(lost, " and ")
}

// holdsPlannedValues reports whether an existing registry already has every value that the plan knows.
// The secrets are not compared: Platform Manager never returns them.
func holdsPlannedValues(ctx context.Context, data *instanceClusterResourceData, existing *webclient.InstanceClusterSchemaRegistry) bool {
	found := *data
	if mapSchemaRegistryResponseToData(ctx, &found, existing).HasError() {
		return false
	}
	foundValues := found.SchemaRegistry.Attributes()
	for name, planned := range data.SchemaRegistry.Attributes() {
		if !planned.IsUnknown() && !planned.Equal(foundValues[name]) {
			return false
		}
	}
	return true
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
	registry.Authentication = stringOrNull(spellingOf(credentials.AuthenticationType, schemaRegistryAuthentications))
	registry.Username = stringOrNull(credentials.Username)
	registry.Certificate = stringOrNull(pemFromAPI(credentials.Certificate, registry.Certificate))
	registry.Truststore = stringOrNull(pemFromAPI(credentials.Truststore, registry.Truststore))
	if registry.Password.IsUnknown() {
		registry.Password = types.StringNull()
	}
	if registry.PrivateKey.IsUnknown() {
		registry.PrivateKey = types.StringNull()
	}
	// A secret that Platform Manager no longer has is planned again: it loses it, for example, when the
	// Instance short_name or the Cluster name changes.
	if lost := lostSecrets(&registry, credentials); lost != "" {
		diags.AddWarning("Schema registry "+lost+" lost",
			fmt.Sprintf("Platform Manager has no %s for the schema registry of Instance-Cluster %s any more, for example because the Instance short_name or the Cluster name changed. The next apply sends it again.",
				lost, data.Id.ValueString()))
	}
	registry.OauthEnabled = types.BoolValue(answer.Authentications != nil && answer.Authentications.OAuth)
	registry.Listeners = stringMapToData(answer.Listeners)
	registry.Properties = stringMapToData(answer.Properties)
	registry.Api = types.StringNull()
	if answer.Api != nil {
		registry.Api = stringOrNull(*answer.Api)
	}
	registry.ClientEndpoints = clientEndpointsToData(answer.ClientEndpoints)

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

package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Lengths and patterns Platform Manager enforces on a Cluster (ClusterConstants, CoreConstants).
const (
	clusterNameMinLength        = 3
	clusterNameMaxLength        = 50
	clusterDescriptionMaxLength = 200
	clusterLocationMaxLength    = 50
	clusterNamePattern          = `^[a-zA-Z0-9._-]+$`
	// clusterPatternPattern is what PM accepts for the topic, group and transactional patterns.
	clusterPatternPattern = `^\{[^}]+\}([-._]\{[^}]+\})*$`
)

var _ resource.Resource = &clusterResource{}
var _ resource.ResourceWithImportState = &clusterResource{}

func NewClusterResource(provider AxualProvider) resource.Resource {
	return &clusterResource{
		provider: provider,
	}
}

type clusterResource struct {
	provider AxualProvider
}

type clusterResourceData struct {
	Id                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	Location                      types.String `tfsdk:"location"`
	Shared                        types.Bool   `tfsdk:"shared"`
	ProviderType                  types.String `tfsdk:"provider_type"`
	Broker                        types.Object `tfsdk:"broker"`
	Strimzi                       types.Object `tfsdk:"strimzi"`
	TopicPattern                  types.String `tfsdk:"topic_pattern"`
	GroupPattern                  types.String `tfsdk:"group_pattern"`
	TransactionalPattern          types.String `tfsdk:"transactional_pattern"`
	KafkaVersion                  types.String `tfsdk:"kafka_version"`
	BootstrapUrlsByAuthentication types.Map    `tfsdk:"bootstrap_urls_by_authentication"`
	KafkaListeners                types.Map    `tfsdk:"kafka_listeners"`
	SslAuthMode                   types.String `tfsdk:"ssl_auth_mode"`
	OauthPrincipalsPrefix         types.String `tfsdk:"oauth_principals_prefix"`
	BillingCloudEnabled           types.Bool   `tfsdk:"billing_cloud_enabled"`
}

type clusterBrokerData struct {
	BootstrapServers types.String `tfsdk:"bootstrap_servers"`
	Protocol         types.String `tfsdk:"protocol"`
	SecurityProtocol types.String `tfsdk:"security_protocol"`
	Certificate      types.String `tfsdk:"certificate"`
	PrivateKey       types.String `tfsdk:"private_key"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	Truststore       types.String `tfsdk:"truststore"`
	CaValidation     types.Bool   `tfsdk:"ca_validation"`
}

type clusterStrimziData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Namespace           types.String `tfsdk:"namespace"`
	ProvisionerUrl      types.String `tfsdk:"provisioner_url"`
	ProvisionerUsername types.String `tfsdk:"provisioner_username"`
	ProvisionerPassword types.String `tfsdk:"provisioner_password"`
}

func clusterBrokerAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"bootstrap_servers": types.StringType,
		"protocol":          types.StringType,
		"security_protocol": types.StringType,
		"certificate":       types.StringType,
		"private_key":       types.StringType,
		"username":          types.StringType,
		"password":          types.StringType,
		"truststore":        types.StringType,
		"ca_validation":     types.BoolType,
	}
}

func clusterStrimziAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cluster_name":         types.StringType,
		"namespace":            types.StringType,
		"provisioner_url":      types.StringType,
		"provisioner_username": types.StringType,
		"provisioner_password": types.StringType,
	}
}

func (r *clusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	patternValidators := []validator.String{
		stringvalidator.RegexMatches(regexp.MustCompile(clusterPatternPattern), "must be placeholders such as {tenant}, {instance}, {environment} and {topic}, joined by -, . or _"),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Cluster is a Kafka cluster that Platform Manager governs. Add it to an Instance with `axual_instance` before topics can be created on it. The broker connection details and credentials are stored in Vault.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Cluster, unique in the tenant. Must be between 3 and 50 characters: letters, digits, `.`, `-` and `_`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(clusterNameMinLength, clusterNameMaxLength),
					stringvalidator.RegexMatches(regexp.MustCompile(clusterNamePattern), "can only contain letters, digits, '.', '-' and '_'"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short description of the Cluster. Must be at most 200 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(clusterDescriptionMaxLength),
				},
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Where the Cluster runs, for example `Amsterdam`. Must be at most 50 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(clusterLocationMaxLength),
				},
			},
			"shared": schema.BoolAttribute{
				MarkdownDescription: "Whether the Cluster is shared by all tenants. A shared Cluster can only be changed and deleted by a Super Admin, so a Tenant Admin keeps the default `false`. Changing it creates a new Cluster.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"provider_type": schema.StringAttribute{
				MarkdownDescription: "How Platform Manager manages the Cluster: `apache-kafka` or `strimzi`. Changing it creates a new Cluster.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("apache-kafka", "strimzi"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"broker": schema.SingleNestedAttribute{
				MarkdownDescription: "How Platform Manager connects to the brokers. Platform Manager checks the connection when the Cluster is created.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"bootstrap_servers": schema.StringAttribute{
						MarkdownDescription: "Bootstrap servers Platform Manager connects to, for example `kafka-0.example.com:9093`.",
						Required:            true,
					},
					"protocol": schema.StringAttribute{
						MarkdownDescription: "How Platform Manager authenticates: `MTLS`, `SASL/PLAIN`, `SASL/SCRAM-SHA-256` or `SASL/SCRAM-SHA-512`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("MTLS", "SASL/PLAIN", "SASL/SCRAM-SHA-256", "SASL/SCRAM-SHA-512"),
						},
					},
					"security_protocol": schema.StringAttribute{
						MarkdownDescription: "Kafka security protocol for SASL: `SASL_SSL` or `SASL_PLAINTEXT`. Platform Manager uses `SASL_SSL` when it is not set.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("SASL_SSL", "SASL_PLAINTEXT"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"certificate": schema.StringAttribute{
						MarkdownDescription: "PEM client certificate (chain) for `MTLS`. Requires `private_key`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("private_key")),
						},
					},
					"private_key": schema.StringAttribute{
						MarkdownDescription: "PEM private key of `certificate`. Platform Manager never returns it, so it is not read back and cannot be imported. This field is Sensitive.",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("certificate")),
						},
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "SASL username. Requires `password`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("password")),
						},
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "SASL password. Platform Manager never returns it, so it is not read back and cannot be imported. This field is Sensitive.",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("username")),
						},
					},
					"truststore": schema.StringAttribute{
						MarkdownDescription: "PEM CA certificates that sign the broker certificates.",
						Optional:            true,
					},
					"ca_validation": schema.BoolAttribute{
						MarkdownDescription: "Whether Platform Manager checks that `truststore` holds a valid CA. Default `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
				},
			},
			"strimzi": schema.SingleNestedAttribute{
				MarkdownDescription: "The Strimzi Kafka cluster behind the Cluster. Required when `provider_type` is `strimzi` (Platform Manager 16.1.0 or later).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"cluster_name": schema.StringAttribute{
						MarkdownDescription: "The `strimzi.io/cluster` name of the Kafka cluster.",
						Required:            true,
					},
					"namespace": schema.StringAttribute{
						MarkdownDescription: "The Kubernetes namespace of the Kafka cluster.",
						Required:            true,
					},
					"provisioner_url": schema.StringAttribute{
						MarkdownDescription: "The Strimzi provisioner URL for this Cluster, when it differs from the one Platform Manager is configured with.",
						Optional:            true,
					},
					"provisioner_username": schema.StringAttribute{
						MarkdownDescription: "Username for the Strimzi provisioner. Requires `provisioner_password`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("provisioner_password")),
						},
					},
					"provisioner_password": schema.StringAttribute{
						MarkdownDescription: "Password for the Strimzi provisioner. Platform Manager never returns it. This field is Sensitive.",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("provisioner_username")),
						},
					},
				},
			},
			"topic_pattern": schema.StringAttribute{
				MarkdownDescription: "How a topic name is made on the Cluster, for example `{tenant}-{instance}-{environment}-{topic}`. Platform Manager refuses a change once topics exist. A Cluster can only serve several environments of one Instance when the pattern contains `{environment}`.",
				Optional:            true,
				Computed:            true,
				Validators:          patternValidators,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_pattern": schema.StringAttribute{
				MarkdownDescription: "How a consumer group id is made on the Cluster, for example `{tenant}-{instance}-{environment}-{group}`.",
				Optional:            true,
				Computed:            true,
				Validators:          patternValidators,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"transactional_pattern": schema.StringAttribute{
				MarkdownDescription: "How a transactional id is made on the Cluster, for example `{tenant}-{instance}-{environment}-{transactional.id}`.",
				Optional:            true,
				Computed:            true,
				Validators:          patternValidators,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kafka_version": schema.StringAttribute{
				MarkdownDescription: "The Kafka version of the Cluster, for information.",
				Optional:            true,
			},
			"bootstrap_urls_by_authentication": schema.MapAttribute{
				MarkdownDescription: "The bootstrap servers applications use, per authentication method: `MUTUAL_TLS` and `SCRAM_SHA_512`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("MUTUAL_TLS", "SCRAM_SHA_512")),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"kafka_listeners": schema.MapAttribute{
				MarkdownDescription: "Listeners of the Cluster, by name.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"ssl_auth_mode": schema.StringAttribute{
				MarkdownDescription: "How the brokers name an SSL client: `certificate` (the subject) or `fullChainCertificate` (the subjects of the whole chain, from the Axual principal builder). With `fullChainCertificate`, an Instance on this Cluster needs `ca_certificates`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("certificate", "fullChainCertificate"),
				},
			},
			"oauth_principals_prefix": schema.StringAttribute{
				MarkdownDescription: "Prefix of OAuth principals on the Cluster. Platform Manager refuses a change while OAuth principals exist.",
				Optional:            true,
			},
			"billing_cloud_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether usage of the Cluster is billed as cloud usage. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data clusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterRequest, diags := createClusterRequestFromData(ctx, &data, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.provider.client.CreateCluster(clusterRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for cluster resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(mapClusterResponseToData(ctx, &data, cluster)...)
	tflog.Info(ctx, "created Cluster")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data clusterResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.provider.client.GetCluster(data.Id.ValueString())
	if err != nil {
		if isGone(err) {
			tflog.Warn(ctx, fmt.Sprintf("Cluster not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Cluster, got error: %s", err))
		}
		return
	}

	resp.Diagnostics.Append(mapClusterResponseToData(ctx, &data, cluster)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data clusterResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterRequest, diags := createClusterRequestFromData(ctx, &data, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.provider.client.UpdateCluster(data.Id.ValueString(), clusterRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Cluster, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(mapClusterResponseToData(ctx, &data, cluster)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data clusterResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := ignoreGone(r.provider.client.DeleteCluster(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Cluster, got error: %s", err))
	}
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importByIDOrKey(ctx, req, resp, importByNamePrefix, func(name string) (string, error) {
		return findClusterIDByName(r.provider.client, name)
	})
}

func findClusterIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("cluster", name, func() ([]keyedID, error) {
		clusters, err := client.GetClustersByName(name)
		if err != nil {
			return nil, err
		}
		found := make([]keyedID, 0, len(clusters))
		for _, cluster := range clusters {
			found = append(found, keyedID{key: cluster.Name, id: cluster.Uid})
		}
		return found, nil
	})
}

// createClusterRequestFromData builds the POST or PATCH body. Platform Manager rewrites the Vault
// secret from providerInfo on every change, so the secrets are always sent again. sharedCluster is
// only sent on create: it can never change.
func createClusterRequestFromData(ctx context.Context, data *clusterResourceData, create bool) (webclient.ClusterRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	var broker clusterBrokerData
	diags.Append(data.Broker.As(ctx, &broker, basetypes.ObjectAsOptions{})...)

	providerInfo := map[string]string{
		webclient.ProviderInfoProviderType:     data.ProviderType.ValueString(),
		webclient.ProviderInfoBootstrapServers: broker.BootstrapServers.ValueString(),
		webclient.ProviderInfoBrokerProtocol:   broker.Protocol.ValueString(),
	}
	putIfSet(providerInfo, webclient.ProviderInfoBrokerSecurity, broker.SecurityProtocol)
	putPEMIfSet(providerInfo, webclient.ProviderInfoBrokerCertificate, broker.Certificate)
	putPEMIfSet(providerInfo, webclient.ProviderInfoBrokerPrivateKey, broker.PrivateKey)
	putIfSet(providerInfo, webclient.ProviderInfoBrokerUsername, broker.Username)
	putIfSet(providerInfo, webclient.ProviderInfoBrokerPassword, broker.Password)
	putPEMIfSet(providerInfo, webclient.ProviderInfoBrokerTruststore, broker.Truststore)
	if !broker.CaValidation.IsNull() && !broker.CaValidation.IsUnknown() {
		providerInfo[webclient.ProviderInfoBrokerCAValidation] = fmt.Sprintf("%t", broker.CaValidation.ValueBool())
	}

	settings := map[string]string{}
	putIfSet(settings, webclient.ClusterSettingSslAuthMode, data.SslAuthMode)
	putIfSet(settings, webclient.ClusterSettingOauthPrincipalsPrefix, data.OauthPrincipalsPrefix)
	if !data.Strimzi.IsNull() && !data.Strimzi.IsUnknown() {
		var strimzi clusterStrimziData
		diags.Append(data.Strimzi.As(ctx, &strimzi, basetypes.ObjectAsOptions{})...)
		putIfSet(settings, webclient.ClusterSettingStrimziClusterName, strimzi.ClusterName)
		putIfSet(settings, webclient.ClusterSettingStrimziNamespace, strimzi.Namespace)
		putIfSet(settings, webclient.ClusterSettingStrimziProvisionerUrl, strimzi.ProvisionerUrl)
		putIfSet(providerInfo, webclient.ProviderInfoStrimziProvisionerId, strimzi.ProvisionerUsername)
		putIfSet(providerInfo, webclient.ProviderInfoStrimziProvisionerPw, strimzi.ProvisionerPassword)
	}

	request := webclient.ClusterRequest{
		Name:                          data.Name.ValueString(),
		Description:                   stringPointerOrNil(data.Description),
		Location:                      stringPointerOrNil(data.Location),
		BillingCloudEnabled:           data.BillingCloudEnabled.ValueBool(),
		ProviderType:                  data.ProviderType.ValueString(),
		ProviderInfo:                  providerInfo,
		TopicPattern:                  stringPointerOrNil(data.TopicPattern),
		GroupPattern:                  stringPointerOrNil(data.GroupPattern),
		TransactionalPattern:          stringPointerOrNil(data.TransactionalPattern),
		KafkaVersion:                  stringPointerOrNil(data.KafkaVersion),
		KafkaListeners:                stringMapFromData(data.KafkaListeners),
		BootstrapUrlsByAuthentication: stringMapFromData(data.BootstrapUrlsByAuthentication),
		Settings:                      settings,
	}
	if create {
		shared := data.Shared.ValueBool()
		request.SharedCluster = &shared
	}
	return request, diags
}

// mapClusterResponseToData writes the API's view into state. The secrets are kept as they were:
// Platform Manager answers them as "". authenticationInfo is empty for a user who may not see it,
// and then the broker settings in state are kept too.
func mapClusterResponseToData(ctx context.Context, data *clusterResourceData, cluster *webclient.ClusterResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Id = types.StringValue(cluster.Uid)
	data.Name = types.StringValue(cluster.Name)
	data.Description = stringOrNull(cluster.Description)
	data.Location = stringOrNull(cluster.Location)
	data.Shared = types.BoolValue(cluster.SharedCluster)
	if cluster.ProviderType != "" {
		data.ProviderType = types.StringValue(cluster.ProviderType)
	}
	data.TopicPattern = stringOrNull(cluster.TopicPattern)
	data.GroupPattern = stringOrNull(cluster.GroupPattern)
	data.TransactionalPattern = stringOrNull(cluster.TransactionalPattern)
	data.KafkaVersion = stringOrNull(cluster.KafkaVersion)
	data.BillingCloudEnabled = types.BoolValue(cluster.BillingCloudEnabled)
	data.BootstrapUrlsByAuthentication = stringMapToData(cluster.BootstrapUrlsByAuthentication)
	data.KafkaListeners = stringMapToData(cluster.KafkaListeners)
	data.SslAuthMode = stringOrNull(cluster.Settings[webclient.ClusterSettingSslAuthMode])
	data.OauthPrincipalsPrefix = stringOrNull(cluster.Settings[webclient.ClusterSettingOauthPrincipalsPrefix])

	info := cluster.AuthenticationInfo
	if len(info) > 0 {
		var broker clusterBrokerData
		if !data.Broker.IsNull() && !data.Broker.IsUnknown() {
			diags.Append(data.Broker.As(ctx, &broker, basetypes.ObjectAsOptions{})...)
		}
		broker.BootstrapServers = types.StringValue(info[webclient.ProviderInfoBootstrapServers])
		broker.Protocol = types.StringValue(info[webclient.ProviderInfoBrokerProtocol])
		broker.SecurityProtocol = stringOrNull(info[webclient.ProviderInfoBrokerSecurity])
		broker.Certificate = stringOrNull(pemFromAPI(info[webclient.ProviderInfoBrokerCertificate], broker.Certificate))
		broker.Username = stringOrNull(info[webclient.ProviderInfoBrokerUsername])
		broker.Truststore = stringOrNull(pemFromAPI(info[webclient.ProviderInfoBrokerTruststore], broker.Truststore))
		// A blank value means validation is on.
		broker.CaValidation = types.BoolValue(!strings.EqualFold(info[webclient.ProviderInfoBrokerCAValidation], "false"))
		if broker.PrivateKey.IsUnknown() {
			broker.PrivateKey = types.StringNull()
		}
		if broker.Password.IsUnknown() {
			broker.Password = types.StringNull()
		}
		object, d := types.ObjectValueFrom(ctx, clusterBrokerAttrTypes(), broker)
		diags.Append(d...)
		data.Broker = object
	}

	strimziName := cluster.Settings[webclient.ClusterSettingStrimziClusterName]
	strimziNamespace := cluster.Settings[webclient.ClusterSettingStrimziNamespace]
	if strimziName == "" && strimziNamespace == "" {
		data.Strimzi = types.ObjectNull(clusterStrimziAttrTypes())
	} else {
		var strimzi clusterStrimziData
		if !data.Strimzi.IsNull() && !data.Strimzi.IsUnknown() {
			diags.Append(data.Strimzi.As(ctx, &strimzi, basetypes.ObjectAsOptions{})...)
		}
		strimzi.ClusterName = types.StringValue(strimziName)
		strimzi.Namespace = types.StringValue(strimziNamespace)
		strimzi.ProvisionerUrl = stringOrNull(cluster.Settings[webclient.ClusterSettingStrimziProvisionerUrl])
		if len(info) > 0 {
			strimzi.ProvisionerUsername = stringOrNull(info[webclient.ProviderInfoStrimziProvisionerId])
		}
		if strimzi.ProvisionerPassword.IsUnknown() {
			strimzi.ProvisionerPassword = types.StringNull()
		}
		object, d := types.ObjectValueFrom(ctx, clusterStrimziAttrTypes(), strimzi)
		diags.Append(d...)
		data.Strimzi = object
	}
	return diags
}

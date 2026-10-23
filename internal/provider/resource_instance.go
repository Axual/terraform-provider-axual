package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Lengths and patterns Platform Manager enforces on an Instance (InstanceConstants, CoreConstants).
const (
	instanceNameMinLength        = 3
	instanceNameMaxLength        = 50
	instanceShortNameMaxLength   = 12
	instanceDescriptionMaxLength = 200
)

// instanceAuthenticationMethods are the mechanisms an Instance can enable. MUTUAL_TLS is SSL; the
// others are SASL.
var instanceAuthenticationMethods = []string{"MUTUAL_TLS", "SCRAM_SHA_256", "SCRAM_SHA_512", "OAUTH_BEARER", "PLAIN"}

var _ resource.Resource = &instanceResource{}
var _ resource.ResourceWithImportState = &instanceResource{}

func NewInstanceResource(provider AxualProvider) resource.Resource {
	return &instanceResource{
		provider: provider,
	}
}

type instanceResource struct {
	provider AxualProvider
}

type instanceResourceData struct {
	Id                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	ShortName                  types.String `tfsdk:"short_name"`
	Description                types.String `tfsdk:"description"`
	Clusters                   types.Set    `tfsdk:"clusters"`
	AuthenticationMethods      types.Set    `tfsdk:"authentication_methods"`
	CaCertificates             types.Set    `tfsdk:"ca_certificates"`
	Properties                 types.Map    `tfsdk:"properties"`
	GranularBrowsePermission   types.Bool   `tfsdk:"granular_browse_permission"`
	TopicOwnerBrowsePermission types.Bool   `tfsdk:"topic_owner_browse_permission"`
	EnvironmentMapping         types.Bool   `tfsdk:"environment_mapping"`
	SelfApproval               types.Bool   `tfsdk:"self_approval"`
	CertificateReusePrevention types.String `tfsdk:"certificate_reuse_prevention"`
	ConnectEnabled             types.Bool   `tfsdk:"connect_enabled"`
	ConnectUrls                types.List   `tfsdk:"connect_urls"`
	FlinkEnabled               types.Bool   `tfsdk:"flink_enabled"`
}

func (r *instanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An Instance is a group of one or more Clusters that get the same topics and ACLs. Environments are mapped to an Instance. Use `axual_instance_cluster` to give a Cluster of the Instance a schema registry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Instance, unique in the tenant. Must be between 3 and 50 characters: letters, digits, spaces, `.`, `-` and `_`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(instanceNameMinLength, instanceNameMaxLength),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9._\- ]+$`), "can only contain letters, digits, spaces, '.', '-' and '_'"),
				},
			},
			"short_name": schema.StringAttribute{
				MarkdownDescription: "The short name of the Instance, unique in the tenant and part of topic names. Lower-case letters and digits, starting with a letter, at most 12 characters. Platform Manager refuses a change while the Instance has environments.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, instanceShortNameMaxLength),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9]*$`), "must be lower-case letters and digits, starting with a letter"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A short description of the Instance. Must be at most 200 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(instanceDescriptionMaxLength),
				},
			},
			"clusters": schema.SetAttribute{
				MarkdownDescription: "IDs of the Clusters in the Instance, at least one. Removing a Cluster deletes its schema registry configuration; Platform Manager refuses it while a Kafka Connect or Flink cluster is registered on it.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"authentication_methods": schema.SetAttribute{
				MarkdownDescription: "How applications may authenticate: `MUTUAL_TLS`, `SCRAM_SHA_256`, `SCRAM_SHA_512`, `OAUTH_BEARER` or `PLAIN`. Each must be enabled for the tenant. Platform Manager enables `MUTUAL_TLS` when it is not set, and refuses removing a method that is in use.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(instanceAuthenticationMethods...)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"ca_certificates": schema.SetAttribute{
				MarkdownDescription: "PEM CA certificates that sign application certificates. Required when `MUTUAL_TLS` is enabled and a Cluster has `ssl_auth_mode` `fullChainCertificate`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"properties": schema.MapAttribute{
				MarkdownDescription: "Defaults for topics on the Instance, for example `replication.factor`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"granular_browse_permission": schema.BoolAttribute{
				MarkdownDescription: "Whether topic browsing is allowed per user and group (granular stream browse permissions). Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"topic_owner_browse_permission": schema.BoolAttribute{
				MarkdownDescription: "Whether topic owners may always browse their topics. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"environment_mapping": schema.BoolAttribute{
				MarkdownDescription: "Whether environments can be mapped to the Instance. Default `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"self_approval": schema.BoolAttribute{
				MarkdownDescription: "Whether the owner of a topic and an application may approve their own access request. Default `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"certificate_reuse_prevention": schema.StringAttribute{
				MarkdownDescription: "Whether one certificate may be used by more than one application principal: `NONE`, `INSTANCE` (not twice in the Instance) or `TENANT` (not twice in the tenant). Default `NONE`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("NONE"),
				Validators: []validator.String{
					stringvalidator.OneOf("NONE", "INSTANCE", "TENANT"),
				},
			},
			"connect_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Connector applications can be deployed on the Instance. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"connect_urls": schema.ListAttribute{
				MarkdownDescription: "URLs of the Axual Connect REST API, for the legacy Axual Connect. Platform Manager calls each one when `connect_enabled` is `true`.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"flink_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Flink SQL applications can be deployed on the Instance. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data instanceResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.provider.client.CreateInstance(createInstanceRequestFromData(&data, r.provider.client.ApiURL))
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for instance resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapInstanceResponseToData(&data, instance)
	tflog.Info(ctx, "created Instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data instanceResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.provider.client.GetInstance(data.Id.ValueString())
	if err != nil {
		if isGone(err) {
			tflog.Warn(ctx, fmt.Sprintf("Instance not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Instance, got error: %s", err))
		}
		return
	}

	mapInstanceResponseToData(&data, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data instanceResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.provider.client.UpdateInstance(data.Id.ValueString(), createInstanceRequestFromData(&data, r.provider.client.ApiURL))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Instance, got error: %s", err))
		return
	}

	mapInstanceResponseToData(&data, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data instanceResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := ignoreGone(r.provider.client.DeleteInstance(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Instance, got error: %s", err))
	}
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importByIDOrKey(ctx, req, resp, importByNamePrefix, func(name string) (string, error) {
		return findInstanceIDByName(r.provider.client, name)
	})
}

func findInstanceIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("instance", name, func() ([]keyedID, error) {
		instance, err := client.GetInstanceByName(name)
		if err != nil {
			return nil, err
		}
		return []keyedID{{key: instance.Name, id: instance.Uid}}, nil
	})
}

// createInstanceRequestFromData builds the POST or PATCH body with the whole desired state: the
// PATCH replaces the clusters, CA certificates, properties and authentication methods as a whole.
func createInstanceRequestFromData(data *instanceResourceData, apiURL string) webclient.InstanceRequest {
	clusters := make([]webclient.InstanceClusterLink, 0, len(data.Clusters.Elements()))
	for _, id := range stringSetFromData(data.Clusters) {
		clusters = append(clusters, webclient.InstanceClusterLink{Cluster: fmt.Sprintf("%s/clusters/%s", apiURL, id)})
	}

	certificates := make([]webclient.InstanceCertificate, 0)
	for _, pem := range stringSetFromData(data.CaCertificates) {
		certificates = append(certificates, webclient.InstanceCertificate{Pem: pem})
	}

	var methods []webclient.InstanceAuthenticationMethod
	for _, mechanism := range stringSetFromData(data.AuthenticationMethods) {
		protocol := "SASL"
		if mechanism == "MUTUAL_TLS" {
			protocol = "SSL"
		}
		methods = append(methods, webclient.InstanceAuthenticationMethod{Protocol: protocol, Mechanism: mechanism})
	}

	var connectUrls *string
	if urls := stringListFromData(data.ConnectUrls); len(urls) > 0 {
		joined := strings.Join(urls, ",")
		connectUrls = &joined
	}

	return webclient.InstanceRequest{
		Name:                            data.Name.ValueString(),
		ShortName:                       data.ShortName.ValueString(),
		Description:                     stringPointerOrNil(data.Description),
		InstanceClusters:                clusters,
		CaCerts:                         certificates,
		Properties:                      stringMapFromData(data.Properties),
		EnabledAuthenticationMethods:    methods,
		ConnectEnabled:                  data.ConnectEnabled.ValueBool(),
		FlinkEnabled:                    data.FlinkEnabled.ValueBool(),
		ConnectUrls:                     connectUrls,
		GranularBrowsePermission:        data.GranularBrowsePermission.ValueBool(),
		TopicOwnerBrowsePermission:      data.TopicOwnerBrowsePermission.ValueBool(),
		EnvironmentMapping:              data.EnvironmentMapping.ValueBool(),
		SelfApproval:                    data.SelfApproval.ValueBool(),
		CertificateReusePreventionLevel: data.CertificateReusePrevention.ValueString(),
	}
}

func mapInstanceResponseToData(data *instanceResourceData, instance *webclient.InstanceResponse) {
	data.Id = types.StringValue(instance.Uid)
	data.Name = types.StringValue(instance.Name)
	data.ShortName = types.StringValue(instance.ShortName)
	data.Description = stringOrNull(instance.Description)

	clusters := make([]string, 0, len(instance.InstanceClusters))
	for _, instanceCluster := range instance.InstanceClusters {
		clusters = append(clusters, instanceCluster.Cluster.Uid)
	}
	data.Clusters = stringSetToData(clusters)

	methods := make([]string, 0, len(instance.EnabledAuthenticationMethods))
	for _, method := range instance.EnabledAuthenticationMethods {
		methods = append(methods, method.Mechanism)
	}
	data.AuthenticationMethods = stringSetToData(methods)

	certificates := make([]string, 0, len(instance.CaCerts))
	for _, certificate := range instance.CaCerts {
		certificates = append(certificates, certificate.Pem)
	}
	data.CaCertificates = stringSetToData(certificates)

	data.Properties = stringMapToData(instance.Properties)
	data.GranularBrowsePermission = types.BoolValue(instance.GranularBrowsePermission)
	data.TopicOwnerBrowsePermission = types.BoolValue(instance.TopicOwnerBrowsePermission)
	data.EnvironmentMapping = types.BoolValue(instance.EnvironmentMapping)
	data.SelfApproval = types.BoolValue(instance.SelfApproval)
	data.ConnectEnabled = types.BoolValue(instance.ConnectEnabled)
	data.FlinkEnabled = types.BoolValue(instance.FlinkEnabled)
	// The answer is a display name such as "None", while a request takes "NONE".
	data.CertificateReusePrevention = types.StringValue(strings.ToUpper(instance.CertificateReusePreventionLevel))

	if instance.ConnectUrls == "" {
		data.ConnectUrls = types.ListNull(types.StringType)
	} else {
		urls := []attr.Value{}
		for _, u := range strings.Split(instance.ConnectUrls, ",") {
			if u = strings.TrimSpace(u); u != "" {
				urls = append(urls, types.StringValue(u))
			}
		}
		data.ConnectUrls = types.ListValueMust(types.StringType, urls)
	}
}

// stringSetFromData returns the set's strings in a fixed order, or nil for a null or unknown set.
func stringSetFromData(value types.Set) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	out := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if s, ok := element.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out = append(out, s.ValueString())
		}
	}
	sort.Strings(out)
	return out
}

func stringSetToData(values []string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

func stringListFromData(value types.List) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	out := make([]string, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		if s, ok := element.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out = append(out, s.ValueString())
		}
	}
	return out
}

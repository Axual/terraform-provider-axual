package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &tenantSettingsResource{}
var _ resource.ResourceWithImportState = &tenantSettingsResource{}
var _ resource.ResourceWithModifyPlan = &tenantSettingsResource{}

func NewTenantSettingsResource(provider AxualProvider) resource.Resource {
	return &tenantSettingsResource{provider: provider}
}

type tenantSettingsResource struct {
	provider AxualProvider
}

func (r *tenantSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_settings"
}

func (r *tenantSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the tenant.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"authentication_methods": schema.ListAttribute{
			MarkdownDescription: "The login methods that the Instances of the tenant may use: `MUTUAL_TLS`, `SCRAM_SHA_256`, `SCRAM_SHA_512`, `OAUTH_BEARER` or `PLAIN`. The order is the rank: the first one is shown first. A method must also be offered by the installation, and Platform Manager refuses removing a method that an Instance uses.",
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.UniqueValues(),
				listvalidator.ValueStringsAre(stringvalidator.OneOf(tenantAuthenticationMethods...)),
			},
			PlanModifiers: []planmodifier.List{keepStateWhenUnsetList{}},
		},
		"history_retention_days": schema.Int64Attribute{
			MarkdownDescription: "How many days the history (audit events) of the tenant is kept. `0` keeps it forever. Platform Manager refuses more than the platform value, 365 by default. `null` when the tenant uses the platform value.",
			Optional:            true,
			Computed:            true,
			Validators:          []validator.Int64{int64validator.AtLeast(0)},
			PlanModifiers:       []planmodifier.Int64{keepStateWhenUnsetInt64{}},
		},
	}
	for _, s := range tenantBoolSettings {
		attrs[s.attr] = schema.BoolAttribute{
			MarkdownDescription: s.description,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{keepStateWhenUnsetBool{}},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "The settings of the tenant of the login. Only the attributes that the configuration sets are managed: an attribute that is left out keeps its value in Platform Manager and shows it. Destroying this resource changes nothing in the tenant.",
		Attributes:          attrs,
	}
}

// tenantSettingsFromState reads the resource attributes from a plan, a state or a configuration.
func tenantSettingsFromState(ctx context.Context, get func(context.Context, path.Path, interface{}) diag.Diagnostics) (types.String, tenantSettingsValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	var id types.String
	values := tenantSettingsValues{Bools: map[string]types.Bool{}}
	diags.Append(get(ctx, path.Root("id"), &id)...)
	diags.Append(get(ctx, path.Root("authentication_methods"), &values.AuthenticationMethods)...)
	diags.Append(get(ctx, path.Root("history_retention_days"), &values.HistoryRetentionDays)...)
	for _, s := range tenantBoolSettings {
		var v types.Bool
		diags.Append(get(ctx, path.Root(s.attr), &v)...)
		values.Bools[s.attr] = v
	}
	return id, values, diags
}

func setTenantSettingsState(ctx context.Context, state *tfsdk.State, id string, values tenantSettingsValues) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(state.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)
	diags.Append(state.SetAttribute(ctx, path.Root("authentication_methods"), values.AuthenticationMethods)...)
	diags.Append(state.SetAttribute(ctx, path.Root("history_retention_days"), values.HistoryRetentionDays)...)
	for _, s := range tenantBoolSettings {
		diags.Append(state.SetAttribute(ctx, path.Root(s.attr), values.Bools[s.attr])...)
	}
	return diags
}

// readTenant returns the tenant of the login and its settings.
func (r *tenantSettingsResource) readTenant() (*webclient.TenantResponse, tenantSettingsValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	tenant, err := r.provider.client.GetTenant()
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read the tenant of the login: %s", err))
		return nil, tenantSettingsValues{}, diags
	}
	runtimeSupported, err := tenantRuntimeSupported(r.provider.client)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read the Platform Manager version: %s", err))
		return nil, tenantSettingsValues{}, diags
	}
	values, d := tenantSettingsFromAnswer(tenant, runtimeSupported)
	diags.Append(d...)
	return tenant, values, diags
}

// apply brings the tenant to the configured values, then reads it back into state.
func (r *tenantSettingsResource) apply(ctx context.Context, config tfsdk.Config, state *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics
	_, configured, d := tenantSettingsFromState(ctx, config.GetAttribute)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	tenant, current, d := r.readTenant()
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	changes, d := computeTenantChanges(ctx, configured, current)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	if !changes.empty() {
		if err := applyTenantChanges(r.provider.client, tenant.Uid, changes); err != nil {
			diags.AddError("Unable to change the settings of tenant "+tenant.ShortName, err.Error())
			return diags
		}
		tenant, current, d = r.readTenant()
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
	}
	diags.Append(setTenantSettingsState(ctx, state, tenant.Uid, current)...)
	return diags
}

func (r *tenantSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(r.apply(ctx, req.Config, &resp.State)...)
}

func (r *tenantSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(r.apply(ctx, req.Config, &resp.State)...)
}

func (r *tenantSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tenant, current, diags := r.readTenant()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !id.IsNull() && id.ValueString() != "" && id.ValueString() != tenant.Uid {
		resp.Diagnostics.AddError("The login belongs to another tenant",
			fmt.Sprintf("This axual_tenant_settings manages tenant %s, but the provider is logged in to tenant %s (%s). An axual_tenant_settings can only manage the tenant of the login.", id.ValueString(), tenant.ShortName, tenant.Uid))
		return
	}
	resp.Diagnostics.Append(setTenantSettingsState(ctx, &resp.State, tenant.Uid, current)...)
}

// Delete changes nothing: the settings have no "empty" state to go back to, and Instances and applications
// depend on them. The plan warns about this (ModifyPlan).
func (r *tenantSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState takes the ID or the short name of the tenant of the login.
func (r *tenantSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tenant, err := r.provider.client.GetTenant()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read the tenant of the login: %s", err))
		return
	}
	if req.ID != tenant.Uid && req.ID != tenant.ShortName {
		resp.Diagnostics.AddError("Not the tenant of the login",
			fmt.Sprintf("%q is not the ID or the short name of the tenant of the login, %s (%s). An axual_tenant_settings can only manage the tenant of the login.", req.ID, tenant.ShortName, tenant.Uid))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(tenant.Uid))...)
}

func (r *tenantSettingsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning("Destroying axual_tenant_settings changes nothing in the tenant",
			"The settings keep their current values in Platform Manager. Terraform only stops managing them.")
		return
	}
	_, configured, diags := tenantSettingsFromState(ctx, req.Config.GetAttribute)
	resp.Diagnostics.Append(diags...)
	_, planned, diags := tenantSettingsFromState(ctx, req.Plan.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.checkRuntimeVersion(configured, &resp.Diagnostics)

	iam, axual := planned.Bools["iam_group_management"], planned.Bools["axual_group_management"]
	if isFalse(iam) && isFalse(axual) {
		resp.Diagnostics.AddAttributeError(path.Root("axual_group_management"), "At least one group management mode must be on",
			"iam_group_management and axual_group_management would both be false, and Platform Manager refuses that. Keep one of them true.")
	}
	if isFalse(iam) && isTrue(configured.Bools["automatic_iam_group_creation"]) {
		resp.Diagnostics.AddAttributeError(path.Root("automatic_iam_group_creation"), "automatic_iam_group_creation needs iam_group_management",
			"Automatic IAM group creation cannot be on while IAM group management is off. Set iam_group_management = true, or automatic_iam_group_creation = false.")
	}
	// Switching IAM off also switches automatic creation off (the provider sends both together).
	if isFalse(iam) && configured.Bools["automatic_iam_group_creation"].IsNull() && !isFalse(planned.Bools["automatic_iam_group_creation"]) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("automatic_iam_group_creation"), types.BoolValue(false))...)
	}
}

// checkRuntimeVersion refuses a configured Connect, KSML or Flink switch on a Platform Manager that has no such
// tenant setting.
func (r *tenantSettingsResource) checkRuntimeVersion(configured tenantSettingsValues, diags *diag.Diagnostics) {
	if r.provider.client == nil {
		return
	}
	for _, s := range tenantBoolSettings {
		if !s.runtime || configured.Bools[s.attr].IsNull() {
			continue
		}
		version, err := r.provider.client.GetPlatformVersion()
		requirePlatformVersion(diags, version, err, path.Root(s.attr), s.attr, tenantRuntimeMajor, tenantRuntimeMinor,
			"There Connect and Flink are switched on per Instance, with connect_enabled and flink_enabled of axual_instance, and KSML needs no switch.")
		return
	}
}

func isFalse(v types.Bool) bool { return !v.IsNull() && !v.IsUnknown() && !v.ValueBool() }
func isTrue(v types.Bool) bool  { return !v.IsNull() && !v.IsUnknown() && v.ValueBool() }

// keepStateWhenUnset plans the state value for an attribute that the configuration leaves out, also when that
// value is null, so a setting that is not managed never shows as "known after apply". On create the value
// stays unknown and is read from Platform Manager.
type keepStateWhenUnsetBool struct{}
type keepStateWhenUnsetInt64 struct{}
type keepStateWhenUnsetList struct{}

const keepStateDescription = "Keeps the current value when the configuration leaves the attribute out."

func (keepStateWhenUnsetBool) Description(context.Context) string { return keepStateDescription }
func (keepStateWhenUnsetBool) MarkdownDescription(context.Context) string {
	return keepStateDescription
}
func (keepStateWhenUnsetInt64) Description(context.Context) string { return keepStateDescription }
func (keepStateWhenUnsetInt64) MarkdownDescription(context.Context) string {
	return keepStateDescription
}
func (keepStateWhenUnsetList) Description(context.Context) string { return keepStateDescription }
func (keepStateWhenUnsetList) MarkdownDescription(context.Context) string {
	return keepStateDescription
}

func keepState(config attr.Value, state tfsdk.State) bool {
	return config.IsNull() && !state.Raw.IsNull()
}

func (keepStateWhenUnsetBool) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if keepState(req.ConfigValue, req.State) {
		resp.PlanValue = req.StateValue
	}
}

func (keepStateWhenUnsetInt64) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if keepState(req.ConfigValue, req.State) {
		resp.PlanValue = req.StateValue
	}
}

func (keepStateWhenUnsetList) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if keepState(req.ConfigValue, req.State) {
		resp.PlanValue = req.StateValue
	}
}

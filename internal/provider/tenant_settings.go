package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The Connect, KSML and Flink switches are tenant settings from Platform Manager 17.0.0 (master reports 16.1).
// Older versions store the keys without any effect.
const tenantRuntimeMajor, tenantRuntimeMinor = 16, 1

const (
	settingIamGroupManagement   = "iamGroupManagementEnabled"
	settingAxualGroupManagement = "axualGroupManagementEnabled"
	settingAutoIamGroupCreation = "autoIAMGroupCreationEnabled"
	settingHistoryRetentionDays = "historyRetentionDays"
)

// tenantAuthenticationMethods are the login methods a tenant can offer, with the same names as axual_instance.
var tenantAuthenticationMethods = instanceAuthenticationMethods

// tenantBoolSetting is one true/false setting of a tenant.
type tenantBoolSetting struct {
	attr string
	key  string
	// def is the value Platform Manager uses when the key is missing.
	def bool
	// runtime: a Connect, KSML or Flink switch, which needs Platform Manager 17.0.0.
	runtime bool
	// group: changed only with POST /tenant/settings/group-management.
	group       bool
	description string
}

var tenantBoolSettings = []tenantBoolSetting{
	{attr: "connect_runtime_enabled", key: "connectRuntimeEnabled", runtime: true,
		description: "Whether the tenant can use Kafka Connect: Connect clusters and Connector applications. Platform Manager 17.0.0 or later; `null` on older versions, where Connect is switched on per Instance."},
	{attr: "ksml_runtime_enabled", key: "ksmlRuntimeEnabled", runtime: true,
		description: "Whether the tenant can use KSML: KSML provisioners and KSML applications. Platform Manager 17.0.0 or later; `null` on older versions."},
	{attr: "flink_runtime_enabled", key: "flinkRuntimeEnabled", runtime: true,
		description: "Whether the tenant can use Flink: Flink clusters and Flink SQL applications. Platform Manager 17.0.0 or later; `null` on older versions, where Flink is switched on per Instance."},
	{attr: "iam_group_management", key: settingIamGroupManagement, group: true,
		description: "Whether groups can be linked to groups of the identity provider (IAM-managed groups)."},
	{attr: "axual_group_management", key: settingAxualGroupManagement, def: true, group: true,
		description: "Whether groups and their members are managed in Axual. At least one of `iam_group_management` and `axual_group_management` must be `true`."},
	{attr: "automatic_iam_group_creation", key: settingAutoIamGroupCreation, group: true,
		description: "Whether a group of the identity provider becomes an Axual group automatically at login. Needs `iam_group_management = true`; it is `false` while IAM group management is off."},
	{attr: "notifications_enabled", key: "notificationsEnabled",
		description: "Whether users get notifications, for example about access requests."},
	{attr: "schema_roles_enforced", key: "schemaRolesEnforced",
		description: "Whether schema roles are enforced on topics."},
	{attr: "data_masking_enabled", key: "dataMaskingEnabled",
		description: "Whether data masking can be used. Needed for the `enforceDataMasking` property of `axual_environment`."},
	{attr: "only_resource_manager_can_update_and_deploy", key: "onlyResourceManagerCanUpdateAndDeploy",
		description: "Whether only the owners of a resource may change and deploy it."},
}

// tenantSettingsValues holds the managed values of a tenant, one field for each attribute of axual_tenant_settings.
type tenantSettingsValues struct {
	AuthenticationMethods types.List
	Bools                 map[string]types.Bool
	HistoryRetentionDays  types.Int64
}

// tenantSettingsFromAnswer reads the settings from GET /tenant. A missing key is read as its Platform Manager
// default, without looking at case. The runtime switches are null when runtimeSupported is false.
func tenantSettingsFromAnswer(tenant *webclient.TenantResponse, runtimeSupported bool) (tenantSettingsValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := tenantSettingsValues{Bools: map[string]types.Bool{}}
	for _, s := range tenantBoolSettings {
		if s.runtime && !runtimeSupported {
			values.Bools[s.attr] = types.BoolNull()
			continue
		}
		values.Bools[s.attr] = types.BoolValue(settingBool(tenant.Settings, s.key, s.def))
	}
	// Platform Manager keeps automatic creation "true" when IAM is switched off without it (an API call with
	// only the IAM switch). It has no effect then, so it reads as false.
	if !values.Bools["iam_group_management"].ValueBool() {
		values.Bools["automatic_iam_group_creation"] = types.BoolValue(false)
	}

	values.HistoryRetentionDays = types.Int64Null()
	if v := settingValue(tenant.Settings, settingHistoryRetentionDays); v != "" {
		days, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			diags.AddWarning("Tenant setting historyRetentionDays is not a number",
				fmt.Sprintf("Platform Manager has %q, so history_retention_days is left empty.", v))
		} else {
			values.HistoryRetentionDays = types.Int64Value(days)
		}
	}

	list, d := types.ListValueFrom(context.Background(), types.StringType, sortedTenantMethods(tenant.SupportedAuthenticationMethods))
	diags.Append(d...)
	values.AuthenticationMethods = list
	return values, diags
}

func settingValue(settings map[string]*string, key string) string {
	if v, ok := settings[key]; ok && v != nil {
		return strings.TrimSpace(*v)
	}
	return ""
}

// settingBool reads a true/false setting like Platform Manager does: only "true", in any case, is true. A missing
// or empty key is def.
func settingBool(settings map[string]*string, key string, def bool) bool {
	v := settingValue(settings, key)
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "true")
}

// sortedTenantMethods returns the mechanisms by rank. Platform Manager returns them in random order; equal ranks
// are sorted by name so the order is stable.
func sortedTenantMethods(methods []webclient.TenantAuthenticationMethod) []string {
	sorted := slices.Clone(methods)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Rank != sorted[j].Rank {
			return sorted[i].Rank < sorted[j].Rank
		}
		return sorted[i].Mechanism < sorted[j].Mechanism
	})
	out := make([]string, 0, len(sorted))
	for _, m := range sorted {
		out = append(out, m.Mechanism)
	}
	return out
}

// tenantMethodsRequest gives each mechanism its position in the list as rank.
func tenantMethodsRequest(mechanisms []string) []webclient.TenantAuthenticationMethod {
	out := make([]webclient.TenantAuthenticationMethod, 0, len(mechanisms))
	for i, m := range mechanisms {
		protocol := "SASL"
		if m == "MUTUAL_TLS" {
			protocol = "SSL"
		}
		out = append(out, webclient.TenantAuthenticationMethod{Rank: i, Protocol: protocol, Mechanism: m})
	}
	return out
}

// tenantChanges are the calls that bring a tenant to the wanted values.
type tenantChanges struct {
	patch webclient.TenantSettingsPatch
	group *webclient.GroupManagementSettings
}

func (c tenantChanges) empty() bool {
	return len(c.patch.Settings) == 0 && c.patch.SupportedAuthenticationMethods == nil && c.group == nil
}

// computeTenantChanges compares the configured values (null = not managed) with the current ones and returns only
// what differs. The group settings go to the group-management call, always with automatic creation next to the
// IAM switch, so automatic creation cannot stay on while IAM is off.
func computeTenantChanges(ctx context.Context, configured, current tenantSettingsValues) (tenantChanges, diag.Diagnostics) {
	var diags diag.Diagnostics
	changes := tenantChanges{patch: webclient.TenantSettingsPatch{Settings: map[string]string{}}}

	for _, s := range tenantBoolSettings {
		want := configured.Bools[s.attr]
		if s.group || want.IsNull() || want.IsUnknown() {
			continue
		}
		if have := current.Bools[s.attr]; have.IsNull() || have.ValueBool() != want.ValueBool() {
			changes.patch.Settings[s.key] = strconv.FormatBool(want.ValueBool())
		}
	}

	if want := configured.HistoryRetentionDays; !want.IsNull() && !want.IsUnknown() {
		if have := current.HistoryRetentionDays; have.IsNull() || have.ValueInt64() != want.ValueInt64() {
			changes.patch.Settings[settingHistoryRetentionDays] = strconv.FormatInt(want.ValueInt64(), 10)
		}
	}

	if want := configured.AuthenticationMethods; !want.IsNull() && !want.IsUnknown() {
		var wantMethods, haveMethods []string
		diags.Append(want.ElementsAs(ctx, &wantMethods, false)...)
		diags.Append(current.AuthenticationMethods.ElementsAs(ctx, &haveMethods, false)...)
		if !slices.Equal(wantMethods, haveMethods) {
			changes.patch.SupportedAuthenticationMethods = tenantMethodsRequest(wantMethods)
		}
	}

	pick := func(attr string) (value bool, managed bool) {
		if v := configured.Bools[attr]; !v.IsNull() && !v.IsUnknown() {
			return v.ValueBool(), true
		}
		return current.Bools[attr].ValueBool(), false
	}
	iam, iamManaged := pick("iam_group_management")
	axual, axualManaged := pick("axual_group_management")
	auto, autoManaged := pick("automatic_iam_group_creation")
	if !iam {
		auto = false
	}
	if iamManaged || axualManaged || autoManaged {
		if iam != current.Bools["iam_group_management"].ValueBool() ||
			axual != current.Bools["axual_group_management"].ValueBool() ||
			auto != current.Bools["automatic_iam_group_creation"].ValueBool() {
			changes.group = &webclient.GroupManagementSettings{
				IamGroupManagement:   &webclient.IamGroupManagement{Enabled: iam, AutomaticGroupCreationEnabled: &auto},
				AxualGroupManagement: &webclient.AxualGroupManagement{Enabled: axual},
			}
		}
	}
	return changes, diags
}

// applyTenantChanges sends the changes one after the other: first the PATCH, then the group-management call.
func applyTenantChanges(client *webclient.Client, tenantId string, changes tenantChanges) error {
	if len(changes.patch.Settings) > 0 || changes.patch.SupportedAuthenticationMethods != nil {
		if err := client.PatchTenantSettings(tenantId, changes.patch); err != nil {
			return fmt.Errorf("changing the tenant settings: %s", platformManagerMessage(err))
		}
	}
	if changes.group != nil {
		if err := client.UpdateGroupManagementSettings(*changes.group); err != nil {
			return fmt.Errorf("changing the group management settings: %s", platformManagerMessage(err))
		}
	}
	return nil
}

// platformManagerMessage returns the message of a Platform Manager error answer, so the user sees "Tenant
// supportedAuthenticationMethods cannot be unselected ..." instead of a JSON body. Platform Manager puts the
// readable text in "detail", and the same text as a list of strings in "errors". Other errors are returned as they are.
func platformManagerMessage(err error) string {
	var httpErr *webclient.HTTPError
	if !errors.As(err, &httpErr) {
		return err.Error()
	}
	var body struct {
		Detail  string            `json:"detail"`
		Message string            `json:"message"`
		Errors  []json.RawMessage `json:"errors"`
	}
	if json.Unmarshal([]byte(httpErr.Body), &body) != nil {
		return err.Error()
	}
	text := body.Detail
	if text == "" {
		var parts []string
		for _, raw := range body.Errors {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				parts = append(parts, s)
			}
		}
		text = strings.Join(parts, " ")
	}
	if text == "" {
		text = body.Message
	}
	if text == "" {
		return err.Error()
	}
	if httpErr.StatusCode == http.StatusForbidden {
		text += ". Changing tenant settings needs the TENANT_ADMIN role"
	}
	return fmt.Sprintf("%s (status %d)", text, httpErr.StatusCode)
}

// tenantRuntimeSupported reads the Platform Manager version and says whether the runtime switches are tenant
// settings there.
func tenantRuntimeSupported(client *webclient.Client) (bool, error) {
	version, err := client.GetPlatformVersion()
	if err != nil {
		return false, err
	}
	return platformVersionAtLeast(version, tenantRuntimeMajor, tenantRuntimeMinor), nil
}

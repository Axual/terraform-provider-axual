package tests

import (
	webclient "axual-webclient"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// tenantPlainSettings are the tenant settings the axual_tenant_settings tests may change through the PATCH, and the
// value Platform Manager uses when a key is missing. historyRetentionDays has no default value: it is put back as
// null, which Platform Manager reads as "the platform value".
var tenantPlainSettings = map[string]string{
	"notificationsEnabled":                  "false",
	"schemaRolesEnforced":                   "false",
	"dataMaskingEnabled":                    "false",
	"onlyResourceManagerCanUpdateAndDeploy": "false",
}

var tenantRuntimeSettings = []string{"connectRuntimeEnabled", "ksmlRuntimeEnabled", "flinkRuntimeEnabled"}

// SaveTenantSettings reads the tenant of the login and puts its settings and login methods back when the test ends,
// also after a failure. A stopped run (Ctrl-C or -timeout) does not put them back.
func SaveTenantSettings(t *testing.T) *webclient.TenantResponse {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	tenant, err := client.GetTenant()
	if err != nil {
		t.Fatalf("unable to read the tenant: %v", err)
	}
	version, err := client.GetPlatformVersion()
	if err != nil {
		t.Fatalf("unable to read the Platform Manager version: %v", err)
	}
	t.Cleanup(func() { restoreTenantSettings(t, client, tenant, versionAtLeast(version, 16, 1)) })
	return tenant
}

func restoreTenantSettings(t *testing.T, client *webclient.Client, saved *webclient.TenantResponse, runtimeSupported bool) {
	original := func(key, def string) string {
		if v, ok := saved.Settings[key]; ok && v != nil && *v != "" {
			return *v
		}
		return def
	}
	settings := map[string]any{}
	for key, def := range tenantPlainSettings {
		settings[key] = original(key, def)
	}
	settings["historyRetentionDays"] = nil
	if v := original("historyRetentionDays", ""); v != "" {
		settings["historyRetentionDays"] = v
	}
	patch := func(body map[string]any) error {
		b, _ := json.Marshal(body)
		return client.RequestAndMap("PATCH", fmt.Sprintf("%s/tenants/%s", client.ApiURL, saved.Uid), strings.NewReader(string(b)), nil, nil)
	}
	if err := patch(map[string]any{"settings": settings, "supportedAuthenticationMethods": saved.SupportedAuthenticationMethods}); err != nil {
		t.Errorf("unable to put the tenant settings and login methods back: %v", err)
	}
	if runtimeSupported {
		// One call per switch: a switch that Platform Manager may not switch off (the tenant had applications of that
		// kind before the test) must not stop the others.
		for _, key := range tenantRuntimeSettings {
			if err := patch(map[string]any{"settings": map[string]any{key: original(key, "false")}}); err != nil {
				log.Printf("[WARN] unable to put %s back to %s: %v", key, original(key, "false"), err)
			}
		}
	}
	iam := strings.EqualFold(original("iamGroupManagementEnabled", "false"), "true")
	auto := iam && strings.EqualFold(original("autoIAMGroupCreationEnabled", "false"), "true")
	axual := strings.EqualFold(original("axualGroupManagementEnabled", "true"), "true")
	if err := client.UpdateGroupManagementSettings(webclient.GroupManagementSettings{
		IamGroupManagement:   &webclient.IamGroupManagement{Enabled: iam, AutomaticGroupCreationEnabled: &auto},
		AxualGroupManagement: &webclient.AxualGroupManagement{Enabled: axual},
	}); err != nil {
		t.Errorf("unable to put the group management settings back: %v", err)
	}
}

// CheckTenantSetting checks the value of a tenant setting in Platform Manager itself, not in the Terraform state.
// want "" means that the key is missing or empty.
func CheckTenantSetting(key, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := apiClient()
		if err != nil {
			return err
		}
		tenant, err := client.GetTenant()
		if err != nil {
			return err
		}
		got := ""
		if v, ok := tenant.Settings[key]; ok && v != nil {
			got = *v
		}
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("tenant setting %s is %q in Platform Manager, expected %q", key, got, want)
		}
		return nil
	}
}

// InstanceMethodInUse returns a login method that the test Instance uses, so a test can try to remove it from the
// tenant. It returns "" when the Instance has none.
func InstanceMethodInUse(t *testing.T, config ProviderConfig) string {
	t.Helper()
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	instance, err := client.GetInstanceByName(config.InstanceName)
	if err != nil {
		t.Fatalf("unable to read instance %s: %v", config.InstanceName, err)
	}
	if len(instance.EnabledAuthenticationMethods) == 0 {
		return ""
	}
	return instance.EnabledAuthenticationMethods[0].Mechanism
}

// UnusedTenantMethod returns a login method of the tenant, other than MUTUAL_TLS, that no Instance of the tenant
// uses, so a test can remove it from the tenant and switch it on again. It returns "" when there is none.
func UnusedTenantMethod(t *testing.T, tenant *webclient.TenantResponse) string {
	t.Helper()
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	items, err := client.ListAll("instances", nil)
	if err != nil {
		t.Fatalf("unable to list the instances: %v", err)
	}
	used := map[string]bool{}
	for _, item := range items {
		instance, err := client.GetInstance(item.String("uid"))
		if err != nil {
			t.Fatalf("unable to read instance %s: %v", item.String("uid"), err)
		}
		for _, m := range instance.EnabledAuthenticationMethods {
			used[m.Mechanism] = true
		}
	}
	for _, m := range tenant.SupportedAuthenticationMethods {
		if m.Mechanism != "MUTUAL_TLS" && !used[m.Mechanism] {
			return m.Mechanism
		}
	}
	return ""
}

// SwitchRuntimeOff switches a Connect, KSML or Flink switch of the tenant off with the API, so a test can switch it
// on again with Terraform. It returns false when Platform Manager refuses, because the tenant uses that runtime.
// SaveTenantSettings puts the old value back.
func SwitchRuntimeOff(t *testing.T, tenant *webclient.TenantResponse, key string) bool {
	t.Helper()
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	err = client.PatchTenantSettings(tenant.Uid, webclient.TenantSettingsPatch{Settings: map[string]string{key: "false"}})
	if err != nil {
		log.Printf("[WARN] Platform Manager does not switch %s off: %v", key, err)
		return false
	}
	return true
}

// PlatformRetentionDays returns the platform value of the audit history retention (GET /audit-settings).
func PlatformRetentionDays(t *testing.T) int {
	t.Helper()
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	var settings struct {
		HistoryRetentionDaysDefault int `json:"historyRetentionDaysDefault"`
	}
	if err := client.RequestAndMap("GET", client.ApiURL+"/audit-settings", nil, nil, &settings); err != nil {
		t.Fatalf("unable to read the audit settings: %v", err)
	}
	return settings.HistoryRetentionDaysDefault
}

// SkipWithInstanceRuntimeSwitches skips a test on a Platform Manager that still switches Connect and Flink per
// Instance: the test Instance answer still has connectEnabled. Master builds from before AXPD-12487 report 16.1 too.
func SkipWithInstanceRuntimeSwitches(t *testing.T, config ProviderConfig) {
	t.Helper()
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	instance, err := client.GetInstanceByName(config.InstanceName)
	if err != nil {
		t.Fatalf("unable to read instance %s: %v", config.InstanceName, err)
	}
	if instance.ConnectEnabled != nil {
		t.Skip("this Platform Manager switches Connect and Flink per Instance, not per tenant")
	}
}

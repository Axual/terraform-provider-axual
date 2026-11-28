package TenantSettingsResource

import (
	webclient "axual-webclient"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const (
	settings = "axual_tenant_settings.test"
	tenant   = "data.axual_tenant.test"
)

// methodsByRank returns the login methods of a tenant in rank order, as the provider shows them.
func methodsByRank(methods []webclient.TenantAuthenticationMethod) []string {
	sorted := slices.Clone(methods)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Rank != sorted[j].Rank {
			return sorted[i].Rank < sorted[j].Rank
		}
		return sorted[i].Mechanism < sorted[j].Mechanism
	})
	out := []string{}
	for _, m := range sorted {
		out = append(out, m.Mechanism)
	}
	return out
}

func hclList(values []string) string {
	return `["` + strings.Join(values, `", "`) + `"]`
}

func checkList(name, attr string, values []string) resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr(name, attr+".#", fmt.Sprint(len(values)))}
	for i, v := range values {
		checks = append(checks, resource.TestCheckResourceAttr(name, fmt.Sprintf("%s.%d", attr, i), v))
	}
	return resource.ComposeTestCheckFunc(checks...)
}

func TestTenantSettingsResource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	saved := SaveTenantSettings(t)
	// The platform value: the audit clean-up then deletes nothing that it would not delete anyway.
	days := fmt.Sprint(PlatformRetentionDays(t))
	methods := methodsByRank(saved.SupportedAuthenticationMethods)
	reordered := slices.Clone(methods)
	slices.Reverse(reordered)

	// A login method that the test Instance uses: Platform Manager refuses removing it from the tenant.
	inUse := InstanceMethodInUse(t, config)
	withoutInUse := slices.DeleteFunc(slices.Clone(methods), func(m string) bool { return m == inUse })

	dataSource := `
data "axual_tenant" "test" {
  depends_on = [axual_tenant_settings.test]
}
`
	full := GetProvider() + `
resource "axual_tenant_settings" "test" {
  authentication_methods                      = ` + hclList(reordered) + `
  notifications_enabled                       = true
  schema_roles_enforced                       = false
  data_masking_enabled                        = true
  only_resource_manager_can_update_and_deploy = false
  history_retention_days                      = ` + days + `
  iam_group_management                        = true
  automatic_iam_group_creation                = true
  axual_group_management                      = true
}
` + dataSource
	// Leaves out data_masking_enabled and the login methods: they are no longer managed and keep their values.
	// Switching IAM off also switches automatic creation off.
	smaller := GetProvider() + `
resource "axual_tenant_settings" "test" {
  notifications_enabled  = false
  history_retention_days = 0
  iam_group_management   = false
}
` + dataSource

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				// The data source alone.
				Config: GetProvider() + `data "axual_tenant" "test" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tenant, "id", saved.Uid),
					resource.TestCheckResourceAttr(tenant, "name", saved.Name),
					resource.TestCheckResourceAttr(tenant, "short_name", saved.ShortName),
					checkList(tenant, "authentication_methods", methods),
					resource.TestCheckResourceAttrSet(tenant, "axual_group_management"),
					resource.TestCheckResourceAttrSet(tenant, "onboarding_completed"),
					CheckFrom(16, 1, resource.TestCheckResourceAttrSet(tenant, "connect_runtime_enabled")),
					CheckBefore(16, 1, resource.TestCheckNoResourceAttr(tenant, "connect_runtime_enabled")),
				),
			},
			{
				Config: GetProvider() + `
resource "axual_tenant_settings" "test" {
  iam_group_management   = false
  axual_group_management = false
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`At least one group management mode must be on`),
			},
			{
				Config: GetProvider() + `
resource "axual_tenant_settings" "test" {
  iam_group_management         = false
  automatic_iam_group_creation = true
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`automatic_iam_group_creation needs iam_group_management`),
			},
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(settings, "id", saved.Uid),
					checkList(settings, "authentication_methods", reordered),
					resource.TestCheckResourceAttr(settings, "notifications_enabled", "true"),
					resource.TestCheckResourceAttr(settings, "data_masking_enabled", "true"),
					resource.TestCheckResourceAttr(settings, "history_retention_days", days),
					resource.TestCheckResourceAttr(settings, "automatic_iam_group_creation", "true"),
					// The data source reads the same values.
					checkList(tenant, "authentication_methods", reordered),
					resource.TestCheckResourceAttr(tenant, "notifications_enabled", "true"),
					resource.TestCheckResourceAttr(tenant, "iam_group_management", "true"),
					resource.TestCheckResourceAttr(tenant, "history_retention_days", days),
					// And Platform Manager has them.
					CheckTenantSetting("notificationsEnabled", "true"),
					CheckTenantSetting("historyRetentionDays", days),
					CheckTenantSetting("autoIAMGroupCreationEnabled", "true"),
				),
			},
			{
				Config: smaller,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(settings, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(settings, "notifications_enabled", "false"),
					resource.TestCheckResourceAttr(settings, "history_retention_days", "0"),
					resource.TestCheckResourceAttr(settings, "iam_group_management", "false"),
					resource.TestCheckResourceAttr(settings, "automatic_iam_group_creation", "false"),
					// No longer in the configuration: they keep their values.
					resource.TestCheckResourceAttr(settings, "data_masking_enabled", "true"),
					checkList(settings, "authentication_methods", reordered),
					CheckTenantSetting("dataMaskingEnabled", "true"),
					CheckTenantSetting("historyRetentionDays", "0"),
					CheckTenantSetting("autoIAMGroupCreationEnabled", "false"),
				),
			},
			{
				ResourceName:      settings,
				ImportState:       true,
				ImportStateId:     saved.ShortName,
				ImportStateVerify: true,
			},
			{
				ResourceName:      settings,
				ImportState:       true,
				ImportStateId:     saved.Uid,
				ImportStateVerify: true,
			},
			{
				ResourceName:  settings,
				ImportState:   true,
				ImportStateId: "not-this-tenant",
				ExpectError:   regexp.MustCompile(`is not the ID or the short name of the tenant of the login`),
			},
			{
				// Platform Manager refuses removing a login method that an Instance uses; nothing changes.
				SkipFunc: func() (bool, error) { return inUse == "" || len(withoutInUse) == 0, nil },
				Config: GetProvider() + `
resource "axual_tenant_settings" "test" {
  authentication_methods = ` + hclList(withoutInUse) + `
}
` + dataSource,
				ExpectError: regexp.MustCompile(`(?s)Unable to change the settings of tenant.*cannot be\s+unselected\s+as\s+it\s+is\s+used`),
			},
			{
				// The Connect, KSML and Flink switches on Platform Manager 17.0.0.
				SkipFunc: SkipStepBefore(16, 1, "the Connect, KSML and Flink switches of a tenant"),
				Config: GetProvider() + `
resource "axual_tenant_settings" "test" {
  connect_runtime_enabled = true
  ksml_runtime_enabled    = true
  flink_runtime_enabled   = true
}
` + dataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(settings, "connect_runtime_enabled", "true"),
					resource.TestCheckResourceAttr(settings, "ksml_runtime_enabled", "true"),
					resource.TestCheckResourceAttr(settings, "flink_runtime_enabled", "true"),
					resource.TestCheckResourceAttr(tenant, "ksml_runtime_enabled", "true"),
					CheckTenantSetting("ksmlRuntimeEnabled", "true"),
				),
			},
			{
				// Older Platform Manager versions have no such switch: the plan fails.
				SkipFunc: SkipStepFrom(16, 1, "the Connect, KSML and Flink switches of a tenant"),
				Config: GetProvider() + `
resource "axual_tenant_settings" "test" {
  connect_runtime_enabled = true
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`connect_runtime_enabled needs Platform Manager 17\.0\.0 or later`),
			},
			{
				// Destroy changes nothing in the tenant: CheckDestroy reads the settings afterwards.
				Destroy: true,
				Config:  smaller,
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			CheckTenantSetting("historyRetentionDays", "0"),
			CheckTenantSetting("dataMaskingEnabled", "true"),
		),
	})
}

// One apply switches a login method on for the tenant and creates an Instance that uses it: depends_on orders them.
// The first step removes that method from the tenant, so the second one really switches it on.
func TestTenantSettingsBeforeInstance(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutKafkaCluster(t, config)
	saved := SaveTenantSettings(t)
	all := methodsByRank(saved.SupportedAuthenticationMethods)
	method := UnusedTenantMethod(t, saved)
	if method == "" {
		t.Skip("every login method of the tenant other than MUTUAL_TLS is used by an Instance, so none can be removed and switched on again")
	}
	without := slices.DeleteFunc(slices.Clone(all), func(m string) bool { return m == method })

	settingsOnly := func(methods []string) string {
		return `
resource "axual_tenant_settings" "test" {
  authentication_methods = ` + hclList(methods) + `
}
`
	}
	withInstance := GetClusterProvider() + settingsOnly(all) + `
resource "axual_cluster" "tenant_settings" {
  name          = "tf-test-tenant-settings-cluster"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  topic_pattern = "{tenant}-{instance}-{environment}-{topic}"
  ssl_auth_mode = "fullChainCertificate"
}

resource "axual_instance" "tenant_settings" {
  name                   = "tf-test-tenant-settings"
  short_name             = "tftenantset"
  clusters               = [axual_cluster.tenant_settings.id]
  authentication_methods = ["MUTUAL_TLS", "` + method + `"]
  ca_certificates        = [file(local.instance_ca_certificate_file)]
  depends_on             = [axual_tenant_settings.test]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetClusterProvider() + settingsOnly(without),
				Check:  checkList(settings, "authentication_methods", without),
			},
			{
				Config: withInstance,
				Check: resource.ComposeTestCheckFunc(
					checkList(settings, "authentication_methods", all),
					resource.TestCheckTypeSetElemAttr("axual_instance.tenant_settings", "authentication_methods.*", method),
				),
			},
			{
				Destroy: true,
				Config:  withInstance,
			},
		},
	})
}

// One apply switches Flink on for the tenant and creates a Flink SQL application, which Platform Manager 17.0.0
// refuses while Flink is off: depends_on orders them.
func TestTenantSettingsBeforeApplication(t *testing.T) {
	SkipBefore(t, 16, 1, "the Connect, KSML and Flink switches of a tenant")
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithInstanceRuntimeSwitches(t, config)
	saved := SaveTenantSettings(t)
	if !SwitchRuntimeOff(t, saved, "flinkRuntimeEnabled") {
		t.Skip("the tenant uses Flink, so Flink cannot be switched off before the test")
	}
	application := `
resource "axual_application" "tenant_settings" {
  name             = "tf-test tenant settings flink"
  application_type = "FLINK_SQL"
  short_name       = "tf_test_tenant_settings_flink"
  application_id   = "tf.test.tenant.settings.flink"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  depends_on       = [axual_tenant_settings.test]
}
`
	off := GetProvider() + `
resource "axual_tenant_settings" "test" {
  flink_runtime_enabled = false
}
`
	on := GetProvider() + `
resource "axual_tenant_settings" "test" {
  flink_runtime_enabled = true
}
` + application
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: off,
				Check:  resource.TestCheckResourceAttr(settings, "flink_runtime_enabled", "false"),
			},
			{
				// Platform Manager refuses a Flink SQL application while Flink is off.
				Config:      off + application,
				ExpectError: regexp.MustCompile(`Flink\s+is\s+disabled\s+for\s+this\s+tenant`),
			},
			{
				Config: on,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(settings, "flink_runtime_enabled", "true"),
					resource.TestCheckResourceAttr("axual_application.tenant_settings", "application_type", "FLINK_SQL"),
				),
			},
			{
				Destroy: true,
				Config:  on,
			},
		},
	})
}

// terraform query lists the tenant of the login, and the generated resource can be imported.
func TestTenantSettingsQuery(t *testing.T) {
	saved := SaveTenantSettings(t)
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				// The provider configuration; the query step below only adds the list block.
				Config: GetProvider() + `data "axual_tenant" "test" {}`,
			},
			{
				Query: true,
				Config: `
list "axual_tenant_settings" "tenant" {
  provider         = axual
  include_resource = true
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("axual_tenant_settings.tenant", 1),
					querycheck.ExpectIdentity("axual_tenant_settings.tenant", map[string]knownvalue.Check{
						"id": knownvalue.StringExact(saved.Uid),
					}),
					querycheck.ExpectResourceKnownValues("axual_tenant_settings.tenant",
						nil,
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("id"), KnownValue: knownvalue.StringExact(saved.Uid)},
							{Path: tfjsonpath.New("authentication_methods"), KnownValue: knownvalue.ListSizeExact(len(saved.SupportedAuthenticationMethods))},
						}),
				},
			},
			{
				// An import block with the identity, like the configuration that terraform query generates.
				Config: GetProvider() + `
import {
  to       = axual_tenant_settings.test
  identity = { id = "` + saved.Uid + `" }
}
resource "axual_tenant_settings" "test" {}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(settings, tfjsonpath.New("id"), knownvalue.StringExact(saved.Uid)),
				},
			},
			{
				Destroy: true,
				Config:  GetProvider() + `resource "axual_tenant_settings" "test" {}`,
			},
		},
	})
}

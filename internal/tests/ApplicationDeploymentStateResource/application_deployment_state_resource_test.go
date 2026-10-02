package ApplicationDeploymentStateResource

import (
	webclient "axual-webclient"
	"fmt"
	"regexp"
	"testing"
	"time"

	. "axual.com/terraform-provider-axual/internal/tests"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	kcTlsState      = "axual_application_deployment_state.kc_tls"
	kcTlsDeployment = "axual_application_deployment.kc_tls"
	kcSaslState     = "axual_application_deployment_state.kc_sasl"
	kcSaslDeploy    = "axual_application_deployment.kc_sasl"
	acTlsState      = "axual_application_deployment_state.ac_tls"
	acTlsDeployment = "axual_application_deployment.ac_tls"
)

// captureId stores resourceName's id, for a later step that has to reach the API directly.
func captureId(resourceName string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		*id = rs.Primary.ID
		return nil
	}
}

// stopOutsideTerraform stops the deployment through the API, the way a user pressing STOP in the
// Self-Service UI would, and waits until it reports itself stopped.
func stopOutsideTerraform(t *testing.T, id *string) func() {
	return func() {
		client, err := ApiClient()
		if err != nil {
			t.Fatalf("building API client: %v", err)
		}
		if err := client.OperateApplicationDeployment(*id, "STOP", webclient.ApplicationDeploymentOperationRequest{Action: "STOP"}); err != nil {
			t.Fatalf("stopping deployment %s: %v", *id, err)
		}
		for i := 0; i < 20; i++ {
			status, err := client.GetApplicationDeploymentStatus(*id)
			if err == nil && status.ConnectorState.State == "Stopped" {
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Fatalf("deployment %s did not stop", *id)
	}
}

func connectTest(t *testing.T) {
	t.Helper()
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutConnectCluster(t, config)
}

// TestApplicationDeploymentStateKafkaConnectMtls runs a KC+TLS connector in the Self-Service UI
// order: the deployment with autostart = false first, then the principal (its key goes straight to
// the Connect cluster's Vault, the target being known), then grant and approval, then the state
// resource. It covers start, stop, a config change while running, drift, import and destroy.
func TestApplicationDeploymentStateKafkaConnectMtls(t *testing.T) {
	connectTest(t)
	var deploymentId string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kcTlsState, "state", "RUNNING"),
					resource.TestCheckResourceAttr(kcTlsState, "current_state", "Running"),
					resource.TestCheckResourceAttrPair(kcTlsState, "application_deployment", kcTlsDeployment, "id"),
					resource.TestCheckResourceAttrPair(kcTlsState, "id", kcTlsDeployment, "id"),
					resource.TestCheckResourceAttr(kcTlsDeployment, "autostart", "false"),
					resource.TestCheckResourceAttrPair(kcTlsDeployment, "target_id", "data.axual_connect_cluster.kc_tls", "id"),
					captureId(kcTlsDeployment, &deploymentId),
				),
			},
			{
				Config: GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_stopped.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kcTlsState, "state", "STOPPED"),
					resource.TestCheckResourceAttr(kcTlsState, "current_state", "Stopped"),
				),
			},
			{
				// A config change on a stopped deployment is PATCHed without starting it, then the
				// state resource starts it with the new config.
				Config: GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment_updated.tf", "kc_tls_state_running.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kcTlsDeployment, "configs.errors.tolerance", "none"),
					resource.TestCheckResourceAttr(kcTlsState, "current_state", "Running"),
				),
			},
			{
				// A config change on a running deployment stops, PATCHes and starts it again. The
				// empty plan after this step proves the state resource still reads RUNNING.
				Config: GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(kcTlsDeployment, "configs.errors.tolerance"),
				),
			},
			{
				// Drift: the connector is stopped outside Terraform. The plan wants it running again.
				PreConfig: stopOutsideTerraform(t, &deploymentId),
				Config:    GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(kcTlsState, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(kcTlsDeployment, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.TestCheckResourceAttr(kcTlsState, "current_state", "Running"),
			},
			{
				ResourceName:      kcTlsState,
				ImportState:       true,
				ImportStateVerify: true,
				Config:            GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
			},
			{
				// autostart is not stored on the platform, so the default is imported.
				ResourceName:            kcTlsDeployment,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"autostart"},
				Config:                  GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
			},
			{
				Destroy: true,
				Config:  GetConnectProvider() + GetFile("kc_tls_setup.tf", "kc_tls_deployment.tf", "kc_tls_state_running.tf"),
			},
		},
	})
}

// TestApplicationDeploymentStateKafkaConnectSasl runs a KC+SASL connector: the credential needs the
// deployment and its target to exist, which autostart = false makes possible in one apply.
func TestApplicationDeploymentStateKafkaConnectSasl(t *testing.T) {
	connectTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kcSaslState, "state", "RUNNING"),
					resource.TestCheckResourceAttr(kcSaslState, "current_state", "Running"),
					resource.TestCheckResourceAttrSet("axual_application_credential.kc_sasl", "id"),
					resource.TestCheckResourceAttrPair(kcSaslDeploy, "target_id", "data.axual_connect_cluster.kc_sasl", "id"),
				),
			},
			{
				Config: GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_stopped.tf"),
				Check:  resource.TestCheckResourceAttr(kcSaslState, "current_state", "Stopped"),
			},
			{
				Config: GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
				Check:  resource.TestCheckResourceAttr(kcSaslState, "current_state", "Running"),
			},
			{
				ResourceName:      kcSaslState,
				ImportState:       true,
				ImportStateVerify: true,
				Config:            GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
			},
			{
				ResourceName:            kcSaslDeploy,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"autostart"},
				Config:                  GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
			},
			{
				// Write-only: never returned by GET, only present in the Create() response.
				ResourceName:            "axual_application_credential.kc_sasl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "auth_provider"},
				Config:                  GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
			},
			{
				Destroy: true,
				Config:  GetConnectProvider() + GetFile("kc_sasl_setup.tf", "kc_sasl_state_running.tf"),
			},
		},
	})
}

// TestApplicationDeploymentStateAxualConnectMtls runs a legacy Axual Connect connector (no
// target_id) with the new resources, the same shape as KC.
func TestApplicationDeploymentStateAxualConnectMtls(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("ac_tls_setup.tf", "ac_tls_state_running.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(acTlsState, "current_state", "Running"),
					// No target is stored; the API reports the synthetic legacy Axual Connect target.
					resource.TestMatchResourceAttr(acTlsDeployment, "target_id", regexp.MustCompile(`^axualconnect-`)),
				),
			},
			{
				ResourceName:      acTlsState,
				ImportState:       true,
				ImportStateVerify: true,
				Config:            GetProvider() + GetFile("ac_tls_setup.tf", "ac_tls_state_running.tf"),
			},
			{
				ResourceName:            acTlsDeployment,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"autostart"},
				Config:                  GetProvider() + GetFile("ac_tls_setup.tf", "ac_tls_state_running.tf"),
			},
			{
				ResourceName:      "axual_application_principal.ac_tls",
				ImportState:       true,
				ImportStateVerify: true,
				// Same as the connector principal tests: the key is write-only, and the API returns the
				// certificate without the file's trailing newline.
				ImportStateVerifyIgnore: []string{"private_key", "active", "principal"},
				Config:                  GetProvider() + GetFile("ac_tls_setup.tf", "ac_tls_state_running.tf"),
			},
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("ac_tls_setup.tf", "ac_tls_state_running.tf"),
			},
		},
	})
}

// TestImportThenApply imports a deployment and its active principal into a state that does not
// have them - the real `terraform import` case - and applies. The apply only fills in what the
// platform does not return (`autostart`, and `active` and `private_key` on the principal). It used
// to fail: the provider activated the already active principal, which Platform Manager answers with
// a 403. It must also not rotate the certificate.
func TestImportThenApply(t *testing.T) {
	connectTest(t)
	var deploymentId, principalId string
	idOf := func(id *string) resource.ImportStateIdFunc {
		return func(*terraform.State) (string, error) { return *id, nil }
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetConnectProvider() + GetFile("import_setup.tf"),
				Check: resource.ComposeTestCheckFunc(
					captureId("axual_application_deployment.imp", &deploymentId),
					captureId("axual_application_principal.imp", &principalId),
				),
			},
			{
				Config: GetConnectProvider() + GetFile("import_forget.tf"),
			},
			{
				ResourceName:       "axual_application_deployment.imp",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  idOf(&deploymentId),
				Config:             GetConnectProvider() + GetFile("import_setup.tf"),
			},
			{
				ResourceName:       "axual_application_principal.imp",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  idOf(&principalId),
				Config:             GetConnectProvider() + GetFile("import_setup.tf"),
			},
			{
				Config: GetConnectProvider() + GetFile("import_setup.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axual_application_principal.imp", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("axual_application_deployment.imp", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_deployment.imp", "autostart", "false"),
					resource.TestCheckResourceAttr("axual_application_principal.imp", "active", "true"),
					func(s *terraform.State) error {
						if got := s.RootModule().Resources["axual_application_principal.imp"].Primary.ID; got != principalId {
							return fmt.Errorf("principal id changed from %s to %s: the certificate was rotated", principalId, got)
						}
						return nil
					},
				),
			},
			{
				Destroy: true,
				Config:  GetConnectProvider() + GetFile("import_setup.tf"),
			},
		},
	})
}

// TestApplicationDeploymentSaslTargetNeedsAutostartFalse: with the default autostart = true, a
// Connector on a SASL_SCRAM Kafka Connect cluster fails up front with a message that says how to
// set it up, instead of a START failure the credential could never fix.
func TestApplicationDeploymentSaslTargetNeedsAutostartFalse(t *testing.T) {
	connectTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetConnectProvider() + GetFile("sasl_autostart_error.tf"),
				ExpectError: regexp.MustCompile("Kafka Connect cluster needs autostart = false"),
			},
		},
	})
}

// TestApplicationDeploymentMtlsTargetNeedsAutostartFalse: an MTLS Kafka Connect cluster is refused
// with autostart = true as well, since a principal created before the deployment has no target yet.
func TestApplicationDeploymentMtlsTargetNeedsAutostartFalse(t *testing.T) {
	connectTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetConnectProvider() + GetFile("kc_tls_autostart_error.tf"),
				ExpectError: regexp.MustCompile("Kafka Connect cluster needs autostart = false"),
			},
		},
	})
}

// TestApplicationDeploymentKafkaConnectAutostartFailsAtPlan: when the application already exists,
// autostart = true on a Kafka Connect target is refused by `terraform plan`, before anything changes.
func TestApplicationDeploymentKafkaConnectAutostartFailsAtPlan(t *testing.T) {
	connectTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetConnectProvider() + GetFile("kc_tls_autostart_error_base.tf"),
			},
			{
				Config:      GetConnectProvider() + GetFile("kc_tls_autostart_error_base.tf", "kc_tls_autostart_plan_deployment.tf"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Kafka Connect cluster needs autostart = false"),
			},
		},
	})
}

func TestApplicationDeploymentStateImportNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				ResourceName:  "axual_application_deployment_state.import_target",
				ImportState:   true,
				ImportStateId: "00000000000000000000000000000000",
				Config: GetProvider() + `
resource "axual_application_deployment_state" "import_target" {
  application_deployment = "00000000000000000000000000000000"
  state                  = "RUNNING"
}
`,
				ExpectError: regexp.MustCompile("Application Deployment Not Found"),
			},
		},
	})
}

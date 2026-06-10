package ApplicationPrincipalResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestApplicationPrincipalResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.tf-test-app-principal", "principal", CertPath("generic_application_3.cer")),
				),
			},
			{
				ResourceName:            "axual_application_principal.tf-test-app-principal",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"principal"},
			},
			{
				// Replace the certificate: new principal is created, old one deleted (no activation for non-Connector)
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.tf-test-app-principal", "principal", CertPath("example_stream_processor.cer")),
					resource.TestCheckNoResourceAttr("axual_application_principal.tf-test-app-principal", "active"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_replaced.tf",
				),
			},
		},
	})
}

// TestApplicationPrincipalCustomActiveNoPermaDiff guards the non-Connector active=true case.
// Activation is only supported for Connector applications, so active=true is a no-op here (the
// provider emits a warning). It must still preserve the value the user wrote: previously the
// provider overwrote active with false on every apply, so config (true) and state (false) never
// matched and every subsequent plan showed a perpetual diff. The implicit post-apply plan that the
// test framework runs after each step must be empty.
func TestApplicationPrincipalCustomActiveNoPermaDiff(t *testing.T) {
	const name = "axual_application_principal.tf-test-app-principal"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_active.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					// The user-written value is preserved, not overwritten with false.
					resource.TestCheckResourceAttr(name, "active", "true"),
				),
			},
			{
				// Re-apply the identical config: must be a no-op. Explicit guard against the perma-diff.
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_active.tf",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(name, "active", "true"),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_active.tf",
				),
			},
		},
	})
}

package ApplicationPrincipalResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestApplicationPrincipalImportThenApplyIsANoop covers importing an existing principal, then
// applying config with the same private_key. private_key can never be read back from the API, so
// ImportState always leaves it "" - re-applying the real value used to look identical to a
// certificate rotation (a destructive create-new/delete-old), even though nothing had changed.
func TestApplicationPrincipalImportThenApplyIsANoop(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_import_target.tf",
				),
			},
			{
				ResourceName:            "axual_application_principal.import_target",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key"},
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_import_target.tf",
				),
			},
			{
				// Re-apply the same config right after import. The plan must be empty - not a
				// rotation (id changing) and not a silently-dropped update.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_import_target.tf",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_import_target.tf",
				),
			},
		},
	})
}

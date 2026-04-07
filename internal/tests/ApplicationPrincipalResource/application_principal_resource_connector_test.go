package ApplicationPrincipalResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationPrincipalConnectorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", "certs/generic_application_1.key"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal", "active", "false"),
				),
			},
			{
				ResourceName:            "axual_application_principal.connector_axual_application_principal",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key"},
			},
			{
				// Replace the certificate: new principal is created, activated, old one deleted
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", "certs/generic_application_2.key"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal", "active", "true"),
				),
			},
			{
				// Verify that deleting an active principal returns a clear error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_replaced.tf",
				),
				ExpectError: regexp.MustCompile("Cannot delete active application principal"),
			},
		},
	})
}

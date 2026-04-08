package ApplicationPrincipalResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationPrincipalOauthbearerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_oauthbearer_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "principal", "example-oauthbearer-principal"),
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "custom", "true"),
				),
			},
			{
				ResourceName:            "axual_application_principal.tf-test-app-principal",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			{
				// Replace the principal value: new principal is created, old one deleted (no activation for non-Connector)
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_oauthbearer_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "principal", "example-oauthbearer-principal-updated"),
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "custom", "true"),
					resource.TestCheckNoResourceAttr("axual_application_principal.tf-test-app-principal", "active"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_oauthbearer_replaced.tf",
				),
			},
		},
	})
}

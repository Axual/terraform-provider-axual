package ApplicationPrincipalResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationPrincipalOauthbearerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_principal_oauthbearer_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "principal", "example-oauthbearer-principal"),
					resource.TestCheckResourceAttr("axual_application_principal.tf-test-app-principal", "custom", "true"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_principal_oauthbearer_replaced.tf"),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.tf-test-app-principal", "principal", "certs/certificate2.crt"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_principal_oauthbearer_replaced.tf"),
			},
		},
	})
}

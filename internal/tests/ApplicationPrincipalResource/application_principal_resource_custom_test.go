package ApplicationPrincipalResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					CheckBodyMatchesFile("axual_application_principal.tf-test-app-principal", "principal", "certs/generic_application_3.cer"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_setup.tf",
					"axual_application_principal_custom_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.tf-test-app-principal", "principal", "certs/example_stream_processor.cer"),
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

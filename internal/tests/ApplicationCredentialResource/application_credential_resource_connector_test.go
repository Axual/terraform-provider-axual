package ApplicationCredentialResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationCredentialConnectorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_credential_custom_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_application_credential.tf-test-app-credential", "environment", "axual_environment.tf-test-env", "id"),
					resource.TestCheckResourceAttrPair("axual_application_credential.tf-test-app-credential", "application", "axual_application.tf-test-app", "id"),
					resource.TestCheckResourceAttrSet("axual_application_credential.tf-test-app-credential", "password"),
					resource.TestCheckResourceAttr("axual_application_credential.tf-test-app-credential", "auth_provider", "apache-kafka"),
					resource.TestCheckResourceAttrWith(
						"axual_application_credential.tf-test-app-credential",
						"username",
						func(val string) error {
							if !strings.Contains(val, "tf_test_apptfdev") {
								return fmt.Errorf("expected username to contain 'tf_test_apptfdev', got: %s", val)
							}
							return nil
						},
					),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_credential_custom_initial.tf"),
			},
		},
	})
}

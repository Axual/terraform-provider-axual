package ApplicationCredentialResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"fmt"
	"regexp"
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
				ResourceName:            "axual_application_credential.tf-test-app-credential",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "auth_provider"},
			},
			{
				Config:      GetProvider() + GetFile("axual_application_credential_custom_update.tf"),
				ExpectError: regexp.MustCompile("API does not allow update of application credential"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_credential_custom_initial.tf"),
			},
		},
	})
}

// TestApplicationCredentialImportNotFound covers importing a nonexistent credential id.
// GET /application_credentials/{id} binds the path variable straight to the domain entity
// server-side, so a nonexistent id fails that binding before the request reaches application
// code, and the API answers a raw 400 ("...converted to null") instead of a clean 404. The
// provider now recognizes that specific response and reports "Application Credential Not Found"
// instead of passing the confusing raw message through - confirmed with both a malformed id and a
// well-formed but nonexistent one, since the API responds identically to both.
func TestApplicationCredentialImportNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				ResourceName:  "axual_application_credential.import_target",
				ImportState:   true,
				ImportStateId: "00000000000000000000000000000000",
				Config:        GetProvider() + GetFile("axual_application_credential_import_target.tf"),
				ExpectError:   regexp.MustCompile("Application Credential Not Found"),
			},
			{
				ResourceName:  "axual_application_credential.import_target",
				ImportState:   true,
				ImportStateId: "not-a-well-formed-uid",
				Config:        GetProvider() + GetFile("axual_application_credential_import_target.tf"),
				ExpectError:   regexp.MustCompile("Application Credential Not Found"),
			},
		},
	})
}

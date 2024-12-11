package EnvironmentResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_environment_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "name", "tf-development"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "color", "#19b9be"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "visibility", "Private"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking", "true"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.testKey", "TestValue"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_environment_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "name", "tf-development1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "description", "This is the terraform testing environment1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "color", "#21ccd2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
					resource.TestCheckResourceAttrPair("axual_environment.tf-test-env", "owners", "axual_group.team-integrations2", "id"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "retention_time", "80000"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckNoResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.testKey", "TestValue"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "viewers.#", "2"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_environment_updated_removed_settings.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "name", "tf-development1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "description", "This is the terraform testing environment1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "color", "#21ccd2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
					resource.TestCheckResourceAttrPair("axual_environment.tf-test-env", "owners", "axual_group.team-integrations2", "id"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "retention_time", "80000"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckNoResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking"),
					resource.TestCheckNoResourceAttr("axual_environment.tf-test-env", "settings.testKey"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "viewers.#", "2"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_environment_updated.tf"),
			},
		},
	})
}

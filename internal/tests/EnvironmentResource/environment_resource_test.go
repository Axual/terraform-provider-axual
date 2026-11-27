package EnvironmentResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// deleteOutsideTerraform deletes the environment with the API, as a user does in Self-Service.
func deleteOutsideTerraform(t *testing.T, name string) func() {
	return func() {
		client, err := ApiClient()
		if err != nil {
			t.Fatalf("building API client: %v", err)
		}
		found, err := client.GetEnvironmentByName(name)
		if err != nil || len(found.Embedded.Environments) != 1 {
			t.Fatalf("finding environment %s: %v", name, err)
		}
		if err := client.DeleteEnvironment(found.Embedded.Environments[0].Uid); err != nil {
			t.Fatalf("deleting environment %s: %v", name, err)
		}
	}
}

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
					resource.TestCheckResourceAttrPair("axual_environment.tf-test-env", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking", "true"),
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
					resource.TestCheckResourceAttrPair("axual_environment.tf-test-env", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "retention_time", "80000"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking", "true"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "viewers.#", "1"),
				),
			},
			{
				// Update non-properties fields (description, retention_time, partitions) while leaving
				// properties and settings unchanged. Verifies an update to other attributes does not
				// drop or alter the existing properties/settings maps.
				Config: GetProvider() + GetFile("axual_environment_updated_keep_properties.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "name", "tf-development1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "description", "This is the terraform testing environment2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "retention_time", "90000"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "partitions", "2"),
					// properties and settings must survive the non-properties update.
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.enforceDataMasking", "true"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_environment_removed_settings_properties.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "name", "tf-development1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "description", "This is the terraform testing environment1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "color", "#21ccd2"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
					resource.TestCheckResourceAttrPair("axual_environment.tf-test-env", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "retention_time", "80000"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "properties.%", "0"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "settings.%", "0"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "viewers.#", "1"),
				),
			},
			{
				// The environment is deleted outside Terraform. The plan creates it again.
				PreConfig: deleteOutsideTerraform(t, "tf-development1"),
				Config:    GetProvider() + GetFile("axual_environment_removed_settings_properties.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axual_environment.tf-test-env", plancheck.ResourceActionCreate),
					},
				},
			},
			{
				ResourceName:      "axual_environment.tf-test-env",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Import by name instead of ID
				ResourceName:      "axual_environment.tf-test-env",
				ImportState:       true,
				ImportStateId:     "name:tf-development1",
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_environment_removed_settings_properties.tf"),
			},
		},
	})

	// Test import with empty properties and settings maps
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_environment_empty_properties_settings.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_environment.tf-test-env-empty-maps", "name", "tf-empty-maps"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env-empty-maps", "properties.%", "0"),
					resource.TestCheckResourceAttr("axual_environment.tf-test-env-empty-maps", "settings.%", "0"),
				),
			},
			{
				ResourceName:      "axual_environment.tf-test-env-empty-maps",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_environment_empty_properties_settings.tf"),
			},
		},
	})
}

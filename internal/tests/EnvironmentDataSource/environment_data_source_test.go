package EnvironmentDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_environment.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "name", "tf-development"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "color", "#19b9be"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "visibility", "Private"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttrPair("data.axual_environment.tf-test-env-imported", "owners", "data.axual_group.test_group", "id"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_environment_not_found.tf"),
				ExpectError: regexp.MustCompile("No Environment resources found with name 'non_existent_resource'"),
			},
		},
	})
}

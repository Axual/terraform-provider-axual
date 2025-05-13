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
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "name", "tf-development"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "color", "#19b9be"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "visibility", "Private"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttrPair("data.axual_environment.tf-test-env-imported-by-name", "owners", "data.axual_group.test_group", "id"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_environment_not_found.tf"),
				ExpectError: regexp.MustCompile("No Environment resources found with name 'non_existent_resource'"),
			},
		},
	})
}

func TestEnvironmentDataSourceGetByShortName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_environment.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "name", "tf-development"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "color", "#19b9be"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "visibility", "Private"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-short-name", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttrPair("data.axual_environment.tf-test-env-imported-by-short-name", "owners", "data.axual_group.test_group", "id"),
				),
			},
		},
	})
}

func TestEnvironmentDataSourceGetByShortNameEmpty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_environment.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "name", "tf-development"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "color", "#19b9be"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "visibility", "Private"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttrPair("data.axual_environment.tf-test-env-imported-by-name-and-short-name", "owners", "data.axual_group.test_group", "id"),
				),
			},
		},
	})
}

func TestEnvironmentDataSourceWithoutNameAndShortName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetProvider() + GetFile("axual_environment_without_name_shortName.tf"),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
		},
	})
}

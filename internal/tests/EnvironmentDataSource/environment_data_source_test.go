package EnvironmentDataSource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

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
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "name", "tf-development"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "short_name", "tfdev"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "description", "This is the terraform testing environment"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "color", "#19b9be"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "visibility", "Private"),
					resource.TestCheckResourceAttr("data.axual_environment.tf-test-env", "authorization_issuer", "Auto"),
					resource.TestCheckResourceAttrPair("data.axual_environment.tf-test-env", "owners", "axual_group.team-integrations", "id"),
				),
			},
		},
	})
}

package GroupDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_group.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "name", "testgroup9999"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "email_address", "test.user@axual.com"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "phone_number", "+6112356789"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "members.#", "1"),
					resource.TestCheckResourceAttrPair("data.axual_group.frontend_developers", "members.0", "axual_user.bob", "id"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_group_not_found.tf"),
				ExpectError: regexp.MustCompile("Resource Not Found: No Group resources found with name 'non_existent_resource'"),
			},
		},
	})
}

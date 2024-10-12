package GroupResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_group_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-integrations", "name", "testgroup9999"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "email_address", "test.user@axual.com"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "phone_number", "+6112356789"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "members.#", "1"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "managers.#", "1"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_group_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-integrations", "name", "updatedgroup9999"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "email_address", "updated.user@axual.com"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "phone_number", "+61123456789"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "members.#", "0"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "managers.#", "0"),
				),
			},
			{
				ResourceName:      "axual_group.team-integrations",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_group_updated.tf"),
			},
		},
	})
}

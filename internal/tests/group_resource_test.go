package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupResource(t *testing.T) {
	providerConfig, err := testProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerConfig.ProtoV6ProviderFactories,
		ExternalProviders:        providerConfig.ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: testAccAxualGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-integrations", "name", "testgroup9999"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "email_address", "test.user@axual.com"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "phone_number", "+6112356789"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "members.#", "1"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "managers.#", "1"),
				),
			},
			{
				Config: testAccAxualGroupConfigUpdated(),
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
		},
	})
}

func testAccAxualGroupConfig() string {
	return testAccProviderConfig() + `
 resource "axual_user" "bob" {
   first_name    = "Bob"
   last_name     = "Foo"
   email_address = "bob.foo@example.com"
   phone_number = "+123456"
   roles         = [
     { name = "APPLICATION_AUTHOR" },
     { name = "ENVIRONMENT_AUTHOR" },
     { name = "STREAM_AUTHOR" }
   ]
 }

resource "axual_group" "team-integrations" {
  name          = "testgroup9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [
    axual_user.bob.id,
  ]
  managers       = [
    axual_user.bob.id,
  ]
}
`
}

func testAccAxualGroupConfigUpdated() string {
	return testAccProviderConfig() + `
resource "axual_group" "team-integrations" {
  name          = "updatedgroup9999"
  phone_number  = "+61123456789"
  email_address = "updated.user@axual.com"
}
`
}

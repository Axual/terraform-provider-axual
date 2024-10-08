package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupDataSource(t *testing.T) {
	providerConfig, err := testProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerConfig.ProtoV6ProviderFactories,
		ExternalProviders:        providerConfig.ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAxualGroupDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "name", "testgroup9999"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "email_address", "test.user@axual.com"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "phone_number", "+6112356789"),
					resource.TestCheckResourceAttr("data.axual_group.frontend_developers", "members.#", "1"),
				),
			},
		},
	})
}

func testAccAxualGroupDataSourceConfig() string {
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

data "axual_group" "frontend_developers" {
  name = axual_group.team-integrations.name
}
`
}

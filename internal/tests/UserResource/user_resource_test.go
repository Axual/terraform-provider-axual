package UserResource

import (
	"fmt"
	"strings"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// userConfig imports the test user by email address (GitHub #114) and sets roles, the roles attribute as HCL.
func userConfig(email string, roles string) string {
	return GetProvider() + fmt.Sprintf(`
import {
  to = axual_user.test_user
  id = "email:%[1]s"
}

resource "axual_user" "test_user" {
  first_name    = "Olivia"
  last_name     = "Walker"
  email_address = "%[1]s"
  %[2]s
}
`, email, roles)
}

// Terraform cannot create users, so the last step keeps the user: it only leaves the Terraform state.
const keepUser = `
removed {
  from = axual_user.test_user
  lifecycle {
    destroy = false
  }
}
`

func TestUserResource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("load test config: %v", err)
	}
	email := config.TestUserEmail
	if email == "" || strings.HasPrefix(email, "YOUR_") {
		t.Skip("testUserEmail is not set in test_config.yaml")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: userConfig(email, `roles = [{ name = "APPLICATION_AUTHOR" }, { name = "STREAM_AUTHOR" }]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_user.test_user", "email_address", email),
					resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "2"),
				),
			},
			{
				// Leaving out roles keeps the roles the user has
				Config: userConfig(email, ""),
				Check:  resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "2"),
			},
			{
				// AXPD-9562: an empty list removes all roles
				Config: userConfig(email, "roles = []"),
				Check:  resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "0"),
			},
			{
				Config: GetProvider() + keepUser,
			},
		},
	})
}

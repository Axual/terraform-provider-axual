package UserResource

import (
	webclient "axual-webclient"
	"fmt"
	"strings"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testUser is the user as it was before the test, so the test can keep its names and restore its roles.
type testUser struct {
	email  string
	fields string // first_name, last_name and the optional fields it has, as HCL
	roles  []webclient.UserRole
}

// readTestUser reads the user from Platform Manager.
func readTestUser(t *testing.T, email string) testUser {
	t.Helper()
	client, err := ApiClient()
	if err != nil {
		t.Fatalf("api client: %v", err)
	}
	found, err := client.FindUserByEmail(email)
	if err != nil || len(found.Embedded.Users) != 1 {
		t.Fatalf("expected one user with email %q, err = %v", email, err)
	}
	user, err := client.GetUser(found.Embedded.Users[0].UID)
	if err != nil {
		t.Fatalf("read user: %v", err)
	}
	fields := fmt.Sprintf("first_name = %q\n  last_name = %q", user.FirstName, user.LastName)
	if middleName, ok := user.MiddleName.(string); ok && middleName != "" {
		fields += fmt.Sprintf("\n  middle_name = %q", middleName)
	}
	if phoneNumber, ok := user.PhoneNumber.(string); ok && phoneNumber != "" {
		fields += fmt.Sprintf("\n  phone_number = %q", phoneNumber)
	}
	var roles []webclient.UserRole
	for _, role := range user.Roles {
		roles = append(roles, webclient.UserRole{Name: role.Name})
	}
	return testUser{email: user.EmailAddress.Email, fields: fields, roles: roles}
}

// rolesHCL is the roles attribute for the given roles.
func rolesHCL(roles []webclient.UserRole) string {
	var names []string
	for _, role := range roles {
		names = append(names, fmt.Sprintf("{ name = %q }", role.Name))
	}
	return "roles = [" + strings.Join(names, ", ") + "]"
}

// userConfig imports the test user by email address and sets roles, the roles attribute as HCL.
func userConfig(user testUser, roles string) string {
	return GetProvider() + fmt.Sprintf(`
import {
  to = axual_user.test_user
  id = "email:%[1]s"
}

resource "axual_user" "test_user" {
  email_address = "%[1]s"
  %[2]s
  %[3]s
}
`, user.email, user.fields, roles)
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
	user := readTestUser(t, email)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: userConfig(user, `roles = [{ name = "APPLICATION_AUTHOR" }, { name = "STREAM_AUTHOR" }]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_user.test_user", "email_address", user.email),
					resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "2"),
				),
			},
			{
				// Leaving out roles keeps the roles the user has
				Config: userConfig(user, ""),
				Check:  resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "2"),
			},
			{
				// An empty list removes all roles
				Config: userConfig(user, "roles = []"),
				Check:  resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", "0"),
			},
			{
				// Puts back the roles the user had before the test
				Config: userConfig(user, rolesHCL(user.roles)),
				Check:  resource.TestCheckResourceAttr("axual_user.test_user", "roles.#", fmt.Sprint(len(user.roles))),
			},
			{
				Config: GetProvider() + keepUser,
			},
		},
	})
}

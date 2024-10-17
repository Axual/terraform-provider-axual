package UserResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_user_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_user.bob", "first_name", "Bob"),
					resource.TestCheckResourceAttr("axual_user.bob", "middle_name", "Bar"),
					resource.TestCheckResourceAttr("axual_user.bob", "last_name", "Foo"),
					resource.TestCheckResourceAttr("axual_user.bob", "email_address", "bob.foo@example.com"),
					resource.TestCheckResourceAttr("axual_user.bob", "phone_number", "+123456"),
					resource.TestCheckResourceAttr("axual_user.bob", "roles.0.name", "APPLICATION_AUTHOR"),
					resource.TestCheckResourceAttr("axual_user.bob", "roles.1.name", "ENVIRONMENT_AUTHOR"),
					resource.TestCheckResourceAttr("axual_user.bob", "roles.2.name", "STREAM_AUTHOR"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_user_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_user.bob", "first_name", "Bob1"),
					resource.TestCheckResourceAttr("axual_user.bob", "middle_name", "Bar1"),
					resource.TestCheckResourceAttr("axual_user.bob", "last_name", "Foo1"),
					resource.TestCheckResourceAttr("axual_user.bob", "email_address", "bob1.foo@example.com"),
					resource.TestCheckResourceAttr("axual_user.bob", "phone_number", "+1234567"),
					resource.TestCheckResourceAttr("axual_user.bob", "roles.0.name", "APPLICATION_AUTHOR"),
					resource.TestCheckResourceAttr("axual_user.bob", "roles.1.name", "SCHEMA_AUTHOR"),
				),
			},
			{
				ResourceName:      "axual_user.bob",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_user_updated.tf"),
			},
		},
	})
}

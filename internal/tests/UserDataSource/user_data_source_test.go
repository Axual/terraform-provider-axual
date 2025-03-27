package InstanceDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestUserDataSource(t *testing.T) {
	config, _ := LoadProviderConfig()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_user.test_user", "email", config.UserEmail),
					resource.TestCheckResourceAttrSet("data.axual_user.test_user", "id"),
					resource.TestCheckResourceAttrSet("data.axual_user.test_user", "email"),
					resource.TestCheckResourceAttrSet("data.axual_user.test_user", "first_name"),
					resource.TestCheckResourceAttrSet("data.axual_user.test_user", "last_name"),
				),
			},
			{
				// Test for error case when user is not found
				Config:      GetProvider() + GetFile("axual_user_email_not_found.tf"),
				ExpectError: regexp.MustCompile("No user found with email"),
			},
			{
				Destroy: true,
				Config:  GetProvider(),
			},
		},
	})
}

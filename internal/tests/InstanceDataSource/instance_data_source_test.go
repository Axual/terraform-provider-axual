package InstanceDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestInstanceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_instance_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.dta", "name", "Dev Test Acceptance"),
					resource.TestCheckResourceAttr("data.axual_instance.dta", "description", "Dev Test Acceptance"),
					resource.TestCheckResourceAttr("data.axual_instance.dta", "short_name", "dta"),
					resource.TestCheckResourceAttr("data.axual_instance.dta", "id", "4b0f204ede6542dfae6bf836f8185c5e"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_not_found.tf"),
				ExpectError: regexp.MustCompile("Unable to read instance by name, got error: resource not found"),
			},
			{
				// Invalid name attribute
				Config:      GetProvider() + GetFile("axual_instance_invalid_name.tf"),
				ExpectError: regexp.MustCompile("Attribute name string length must be between 3 and 50, got: 2"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_instance_initial.tf"),
			},
		},
	})
}

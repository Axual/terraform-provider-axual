package InstanceDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestInstanceDataSource(t *testing.T) {
	config, _ := LoadProviderConfig()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.test_instance", "name", config.InstanceName),
				),
			},
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_by_short_name", "name", config.InstanceName),
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_by_short_name", "short_name", config.InstanceShortName),
				),
			},
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_by_name_and_short_name", "name", config.InstanceName),
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_by_name_and_short_name", "short_name", config.InstanceShortName),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_not_found.tf"),
				ExpectError: regexp.MustCompile("Instance not found"),
			},
			{
				// Invalid name attribute
				Config:      GetProvider() + GetFile("axual_instance_invalid_name.tf"),
				ExpectError: regexp.MustCompile("Attribute name string length must be between 3 and 50, got: 2"),
			},
			{
				// Instance without name or shortNAme
				Config:      GetProvider() + GetFile("axual_instance_without_name_shortName.tf"),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
			{
				Destroy: true,
				Config:  GetProvider(),
			},
		},
	})
}

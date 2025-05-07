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
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name", "name", config.InstanceName),
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name", "short_name", config.InstanceShortName),
				),
			},
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name_null_or_empty", "name", config.InstanceName),
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name_null_or_empty", "short_name", config.InstanceShortName),
				),
			},
			{
				Config: GetProvider(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name_null_or_empty", "name", config.InstanceName),
					resource.TestCheckResourceAttr("data.axual_instance.test_instance_short_name_null_or_empty", "short_name", config.InstanceShortName),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_not_found.tf"),
				ExpectError: regexp.MustCompile("Instance not found"),
			},
			{
				// Invalid name attribute
				Config:      GetProvider() + GetFile("axual_instance_invalid_name.tf"),
				ExpectError: regexp.MustCompile("Name must be between 3 and 50 characters"),
			},
			{
				// Instance without name or shortNAme
				Config:      GetProvider() + GetFile("axual_instance_without_name_shortName.tf"),
				ExpectError: regexp.MustCompile("Either `name` or `short_name` must be specified"),
			},
			{
				Destroy: true,
				Config:  GetProvider(),
			},
		},
	})
}

func TestInstanceDataSourceNameValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetProvider() + GetFile("axual_instance_name_empty_without_shortname.tf"),
				ExpectError: regexp.MustCompile("Either `name` or `short_name` must be specified"),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_name_invalid_length.tf"),
				ExpectError: regexp.MustCompile("Name must be between 3 and 50 characters"),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_name_invalid_pattern.tf"),
				ExpectError: regexp.MustCompile("Invalid Name Format"), //Matching the summary instead of details because the message is too long
			},
		},
	})
}

func TestInstanceDataSourceShortNameValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetProvider() + GetFile("axual_instance_shortname_empty_without_name.tf"),
				ExpectError: regexp.MustCompile("Either `name` or `short_name` must be specified"),
			},
			{
				Config:      GetProvider() + GetFile("axual_instance_shortname_invalid_pattern.tf"),
				ExpectError: regexp.MustCompile("Invalid ShortName Format"), //Matching the summary instead of details because the message is too long
			},
		},
	})
}

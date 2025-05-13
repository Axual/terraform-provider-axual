package ApplicationDataSource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "name", "tf-test-app"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "application_type", "Custom"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "short_name", "tf_test_app_short"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttrPair("data.axual_application.tf-test-app-imported-by-name", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "type", "Java"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "visibility", "Public"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-name", "description", "Axual's TF Test Application"),
				),
			},
		},
	})
}

func TestApplicationDataSourceByShortName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "name", "tf-test-app"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "application_type", "Custom"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "short_name", "tf_test_app_short"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttrPair("data.axual_application.tf-test-app-imported-by-short-name", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "type", "Java"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "visibility", "Public"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name", "description", "Axual's TF Test Application"),
				),
			},
		},
	})
}

func TestConnectorApplicationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_connector_application.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "name", "tf-test-app"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "application_type", "Connector"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "application_class", "org.apache.kafka.connect.axual.utils.LogSourceConnector"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "short_name", "tf_test_app"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttrPair("data.axual_application.tf-test-app-imported", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "type", "SOURCE"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "visibility", "Public"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported", "description", "Axual's TF Test Application"),
				),
			},
		},
	})
}

func TestApplicationDataSourceByShortNameAndName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "name", "tf-test-app"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "application_type", "Custom"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "short_name", "tf_test_app_short"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttrPair("data.axual_application.tf-test-app-imported-by-short-name-and-name", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "type", "Java"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "visibility", "Public"),
					resource.TestCheckResourceAttr("data.axual_application.tf-test-app-imported-by-short-name-and-name", "description", "Axual's TF Test Application"),
				),
			},
		},
	})
}

func TestApplicationDataSourceWithoutNameAndShortName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:      GetProvider() + GetFile("axual_application_without_name_shortName.tf"),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
		},
	})
}

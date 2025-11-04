package ApplicationResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "name", "tf-test app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "short_name", "tf_test_app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "type", "Java"),
					resource.TestCheckResourceAttrPair("axual_application.tf-test-app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "description", "Axual's TF Test Application"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "name", "tf-test-app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "short_name", "tf_test_app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_id", "tf.test.app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "type", "Pega"),
					resource.TestCheckResourceAttrPair("axual_application.tf-test-app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "visibility", "Private"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "description", "Axual's TF Test Application1"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_application_invalid_uppercase.tf"),
				ExpectError: regexp.MustCompile(`can only contain lowercase letters, numbers, and\s+underscores and cannot begin with an underscore`),
			},
			{
				ResourceName:      "axual_application.tf-test-app",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_updated.tf"),
			},
		},
	})
}

func TestApplicationResourceAllTypes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Test Java type
			{
				Config: GetProvider() + GetFile("axual_application_type_java.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "Java"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test Kafka Streams type
			{
				Config: GetProvider() + GetFile("axual_application_type_kafka_streams.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "Kafka Streams"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test Pega type
			{
				Config: GetProvider() + GetFile("axual_application_type_pega.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "Pega"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test SAP type
			{
				Config: GetProvider() + GetFile("axual_application_type_sap.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "SAP"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test DotNet type
			{
				Config: GetProvider() + GetFile("axual_application_type_dotnet.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "DotNet"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test Bridge type
			{
				Config: GetProvider() + GetFile("axual_application_type_bridge.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "Bridge"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test Python type
			{
				Config: GetProvider() + GetFile("axual_application_type_python.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "Python"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test KSML type
			{
				Config: GetProvider() + GetFile("axual_application_type_ksml.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "type", "KSML"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-type", "application_type", "Custom"),
				),
			},
			// Test SINK type (Connector)
			{
				Config: GetProvider() + GetFile("axual_application_type_sink.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-connector", "type", "SINK"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-connector", "application_type", "Connector"),
				),
			},
			// Test SOURCE type (Connector)
			{
				Config: GetProvider() + GetFile("axual_application_type_source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app-connector", "type", "SOURCE"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app-connector", "application_type", "Connector"),
				),
			},
			// Test invalid type - should fail validation
			{
				Config:      GetProvider() + GetFile("axual_application_type_invalid.tf"),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_type_source.tf"),
			},
		},
	})
}

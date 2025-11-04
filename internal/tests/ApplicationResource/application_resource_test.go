package ApplicationResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

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
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "name", "tf-test app"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "short_name", "tf_test_app"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "type", "Java"),
					resource.TestCheckResourceAttrPair("axual_application.tf_test_app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "description", "Axual's TF Test Application"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "name", "tf-test-app1"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "short_name", "tf_test_app1"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "application_id", "tf.test.app1"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "type", "Pega"),
					resource.TestCheckResourceAttrPair("axual_application.tf_test_app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "visibility", "Private"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app", "description", "Axual's TF Test Application1"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_application_invalid_uppercase.tf"),
				ExpectError: regexp.MustCompile(`can only contain lowercase letters, numbers, and\s+underscores and cannot begin with an underscore`),
			},
			{
				ResourceName:      "axual_application.tf_test_app",
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
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_java", "type", "Java"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_java", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_java",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test Kafka Streams type
			{
				Config: GetProvider() + GetFile("axual_application_type_kafka_streams.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_kafka_streams", "type", "Kafka Streams"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_kafka_streams", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_kafka_streams",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test Pega type
			{
				Config: GetProvider() + GetFile("axual_application_type_pega.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_pega", "type", "Pega"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_pega", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_pega",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test SAP type
			{
				Config: GetProvider() + GetFile("axual_application_type_sap.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_sap", "type", "SAP"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_sap", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_sap",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test DotNet type
			{
				Config: GetProvider() + GetFile("axual_application_type_dotnet.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_dotnet", "type", "DotNet"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_dotnet", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_dotnet",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test Bridge type
			{
				Config: GetProvider() + GetFile("axual_application_type_bridge.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_bridge", "type", "Bridge"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_bridge", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_bridge",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test Python type
			{
				Config: GetProvider() + GetFile("axual_application_type_python.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_python", "type", "Python"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_python", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_python",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test KSML type
			{
				Config: GetProvider() + GetFile("axual_application_type_ksml.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_ksml", "type", "KSML"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_ksml", "application_type", "Custom"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_ksml",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test SINK type (Connector)
			{
				Config: GetProvider() + GetFile("axual_application_type_sink.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_sink", "type", "SINK"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_sink", "application_type", "Connector"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_sink",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test SOURCE type (Connector)
			{
				Config: GetProvider() + GetFile("axual_application_type_source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_source", "type", "SOURCE"),
					resource.TestCheckResourceAttr("axual_application.tf_test_app_type_source", "application_type", "Connector"),
				),
			},
			{
				ResourceName:      "axual_application.tf_test_app_type_source",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test invalid type - should fail validation
			{
				Config:      GetProvider() + GetFile("axual_application_type_invalid.tf"),
				ExpectError: regexp.MustCompile(`Attribute type value must be one of: `),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_type_java.tf",
					"axual_application_type_kafka_streams.tf",
					"axual_application_type_pega.tf",
					"axual_application_type_dotnet.tf",
					"axual_application_type_bridge.tf",
					"axual_application_type_python.tf",
					"axual_application_type_ksml.tf",
					"axual_application_type_sink.tf",
					"axual_application_type_source.tf",
				),
			},
		},
	})
}

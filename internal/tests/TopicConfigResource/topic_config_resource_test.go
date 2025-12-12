package TopicConfigResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTopicConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_setup.tf", "axual_topic_config_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "retention_time", "864000"),
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "properties.segment.ms", "600012"),
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "properties.retention.bytes", "-1"),
					resource.TestCheckResourceAttrPair("axual_topic_config.tf-topic-config", "topic", "axual_topic.tf-test-topic", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.tf-topic-config", "environment", "axual_environment.tf-test-env", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_setup.tf", "axual_topic_config_updated.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "retention_time", "864001"),
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "properties.segment.ms", "600013"),
					resource.TestCheckResourceAttr("axual_topic_config.tf-topic-config", "properties.retention.bytes", "1"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_setup.tf", "axual_topic_config_properties_removed.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("axual_topic_config.tf-topic-config", "properties"),
				),
			},
			{
				ResourceName:      "axual_topic_config.tf-topic-config",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_topic_config_setup.tf", "axual_topic_config_properties_removed.tf",
				),
			},
		},
	})
}

func TestTopicConfigAvroResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_avro_setup.tf",
					"axual_topic_config_avro_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "partitions", "1"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "retention_time", "864000"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.segment.ms", "600012"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.retention.bytes", "-1"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "key_schema_version", "axual_schema_version.test_key_v1", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "value_schema_version", "axual_schema_version.test_value_v1", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_avro_setup.tf",
					"axual_topic_config_avro_updated.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "retention_time", "864001"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.segment.ms", "600013"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.retention.bytes", "2"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "key_schema_version", "axual_schema_version.test_key_v2", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "value_schema_version", "axual_schema_version.test_value_v2", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_avro_setup.tf",
					"axual_topic_config_incompatible_avro_updated.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "retention_time", "864001"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.segment.ms", "600013"),
					resource.TestCheckResourceAttr("axual_topic_config.example-with-schema-version", "properties.retention.bytes", "2"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "key_schema_version", "axual_schema_version.test_key_v3", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.example-with-schema-version", "value_schema_version", "axual_schema_version.test_value_v2", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile(
					"axual_topic_config_avro_setup.tf",
					"axual_topic_config_avro_properties_removed.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("axual_topic_config.example-with-schema-version", "properties"),
				),
			},
			//TODO: Regular topic import works, but if topic is AVRO topic then import does not work
			//{
			//	ResourceName:      "axual_topic_config.example-with-schema-version",
			//	ImportState:       true,
			//	ImportStateVerify: true,
			//},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_topic_config_avro_setup.tf",
					"axual_topic_config_avro_properties_removed.tf",
				),
			},
		},
	})
}

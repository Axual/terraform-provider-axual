package TopicDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTopicDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_topic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "name", "test-avro-topic"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "description", "Demo of deploying a topic via Terraform"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "key_type", "AVRO"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "value_type", "AVRO"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "retention_policy", "delete"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test-imported", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test-imported", "owners", "axual_topic.topic-avro-test", "owners"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test-imported", "id", "axual_topic.topic-avro-test", "id"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test-imported", "key_schema", "axual_topic.topic-avro-test", "key_schema"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test-imported", "value_schema", "axual_topic.topic-avro-test", "value_schema"),

					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "name", "test-mix-topic"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "description", "Demo of deploying a mixed schema topic via Terraform"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "key_type", "JSON_SCHEMA"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "value_type", "PROTOBUF"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "retention_policy", "compact,delete"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "properties.propertyKey1", "propertyValue3"),
					resource.TestCheckResourceAttr("data.axual_topic.topic_mix_test_imported", "properties.propertyKey2", "propertyValue4"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic_mix_test_imported", "owners", "axual_topic.topic_mix_test", "owners"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic_mix_test_imported", "id", "axual_topic.topic_mix_test", "id"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic_mix_test_imported", "key_schema", "axual_topic.topic_mix_test", "key_schema"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic_mix_test_imported", "value_schema", "axual_topic.topic_mix_test", "value_schema"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_topic_not_found.tf"),
				ExpectError: regexp.MustCompile("No Topic resources found with name 'non_existent_resource'"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_topic.tf"),
			},
		},
	})
}

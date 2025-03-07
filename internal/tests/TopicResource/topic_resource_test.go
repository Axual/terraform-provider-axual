package TopicResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTopicResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_string_topic_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic.topic-test", "name", "test-topic"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "key_type", "String"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "value_type", "String"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "retention_policy", "delete"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "description", "Demo of deploying a topic via Terraform"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "viewers.#", "1"),
				),
			},
			{
				ResourceName:      "axual_topic.topic-test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_string_topic_initial.tf"),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_avro_topic_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic.topic-test", "name", "test-topic"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "key_type", "AVRO"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "value_type", "AVRO"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "retention_policy", "delete"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "description", "Demo of deploying a topic via Terraform"),
					resource.TestCheckResourceAttr("axual_topic.topic-test", "viewers.#", "1"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_avro_topic_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic.topic-test", "description", "Changed Demo of deploying a topic via Terraform"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_avro_topic_properties_removed.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("axual_topic.topic-test", "properties"),
				),
			},
			{
				ResourceName:      "axual_topic.topic-test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_avro_topic_properties_removed.tf"),
			},
		},
	})
}

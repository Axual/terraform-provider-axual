package TopicDataSource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

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
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "name", "test-topic"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "description", "Demo of deploying a topic via Terraform"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "key_type", "AVRO"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "value_type", "AVRO"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "retention_policy", "delete"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "properties.propertyKey1", "propertyValue1"),
					resource.TestCheckResourceAttr("data.axual_topic.topic-test", "properties.propertyKey2", "propertyValue2"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test", "owners", "axual_topic.topic-test", "owners"),
					resource.TestCheckResourceAttrPair("data.axual_topic.topic-test", "id", "axual_topic.topic-test", "id"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_topic.tf"),
			},
		},
	})
}

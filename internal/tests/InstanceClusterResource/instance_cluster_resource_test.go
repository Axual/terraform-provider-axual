package InstanceClusterResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const name = "axual_instance_cluster.tf_test_ic"

func TestInstanceClusterResource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutSchemaRegistry(t, config)

	setup := "axual_instance_cluster_setup.tf"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config:      GetClusterProvider() + GetFile(setup, "axual_instance_cluster_username_without_password.tf"),
				ExpectError: regexp.MustCompile(`password`),
			},
			{
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(name, "instance", "axual_instance.tf_test_ic_instance", "id"),
					resource.TestCheckResourceAttrPair(name, "cluster", "axual_cluster.tf_test_ic_cluster", "id"),
					resource.TestCheckResourceAttr(name, "schema_registry.type", "APICURIO"),
					resource.TestCheckResourceAttr(name, "schema_registry.url", config.SchemaRegistryUrl),
					resource.TestCheckResourceAttr(name, "schema_registry.authentication", "Basic-Auth"),
					resource.TestCheckResourceAttr(name, "schema_registry.username", config.SchemaRegistryUsername),
					resource.TestCheckResourceAttr(name, "schema_registry.oauth_enabled", "false"),
					resource.TestCheckNoResourceAttr(name, "schema_registry.version"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_confluent.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.type", "CONFLUENT"),
					resource.TestCheckResourceAttr(name, "schema_registry.authentication", "No-Auth"),
					resource.TestCheckResourceAttr(name, "schema_registry.version", "7.0"),
					resource.TestCheckNoResourceAttr(name, "schema_registry.username"),
				),
			},
			{
				// An AVRO topic on an environment of the Instance registers its schemas in the
				// registry of the Instance-Cluster.
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.type", "APICURIO"),
					resource.TestCheckResourceAttr(name, "schema_registry.version", "3.0"),
					resource.TestCheckResourceAttrSet("axual_topic_config.tf_test_ic_topic_config", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.tf_test_ic_topic_config", "value_schema_version", "axual_schema_version.tf_test_ic_value", "id"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// Platform Manager never returns the password, so import cannot restore it.
				ImportStateVerifyIgnore: []string{"schema_registry.password"},
				Config:                  GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
			},
		},
	})
}

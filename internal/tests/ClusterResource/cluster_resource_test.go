package ClusterResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const name = "axual_cluster.tf_test_cluster"

func TestClusterResource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutKafkaCluster(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config:      GetClusterProvider() + GetFile("axual_cluster_invalid_provider.tf"),
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
			{
				Config:      GetClusterProvider() + GetFile("axual_cluster_certificate_without_key.tf"),
				ExpectError: regexp.MustCompile(`private_key`),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_cluster_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "name", "tf-test-cluster"),
					resource.TestCheckResourceAttr(name, "description", "Axual's TF Test Cluster"),
					resource.TestCheckResourceAttr(name, "provider_type", "apache-kafka"),
					resource.TestCheckResourceAttr(name, "shared", "false"),
					resource.TestCheckResourceAttr(name, "broker.bootstrap_servers", config.KafkaBootstrapServers),
					resource.TestCheckResourceAttr(name, "broker.protocol", "MTLS"),
					resource.TestCheckResourceAttr(name, "broker.ca_validation", "false"),
					resource.TestCheckResourceAttrSet(name, "broker.certificate"),
					resource.TestCheckResourceAttrSet(name, "broker.truststore"),
					resource.TestCheckResourceAttr(name, "ssl_auth_mode", "fullChainCertificate"),
					resource.TestCheckResourceAttr(name, "billing_cloud_enabled", "false"),
					resource.TestCheckNoResourceAttr(name, "location"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_cluster_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-test-cluster-renamed"),
					resource.TestCheckResourceAttr(name, "description", "Axual's TF Test Cluster, updated"),
					resource.TestCheckResourceAttr(name, "location", "Utrecht"),
					resource.TestCheckResourceAttr(name, "topic_pattern", "{tenant}-{instance}-{environment}-{topic}"),
					resource.TestCheckResourceAttr(name, "group_pattern", "{tenant}-{instance}-{environment}-{group}"),
					resource.TestCheckResourceAttr(name, "transactional_pattern", "{tenant}-{instance}-{environment}-{transactional.id}"),
					resource.TestCheckResourceAttr(name, "kafka_version", "4.2.0"),
					resource.TestCheckResourceAttr(name, "bootstrap_urls_by_authentication.MUTUAL_TLS", config.KafkaBootstrapServers),
					resource.TestCheckResourceAttr(name, "kafka_listeners.MUTUAL_TLS", config.KafkaBootstrapServers),
					resource.TestCheckResourceAttr(name, "ssl_auth_mode", "certificate"),
					resource.TestCheckResourceAttr(name, "oauth_principals_prefix", "tf-test-"),
					resource.TestCheckResourceAttr(name, "billing_cloud_enabled", "true"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_cluster_minimal.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "description"),
					resource.TestCheckNoResourceAttr(name, "location"),
					resource.TestCheckNoResourceAttr(name, "kafka_version"),
					resource.TestCheckNoResourceAttr(name, "ssl_auth_mode"),
					resource.TestCheckNoResourceAttr(name, "oauth_principals_prefix"),
					resource.TestCheckResourceAttr(name, "billing_cloud_enabled", "false"),
					// The patterns are optional and computed, so leaving them out keeps them.
					resource.TestCheckResourceAttr(name, "topic_pattern", "{tenant}-{instance}-{environment}-{topic}"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// Platform Manager never returns the private key, so import cannot restore it.
				ImportStateVerifyIgnore: []string{"broker.private_key"},
				Config:                  GetClusterProvider() + GetFile("axual_cluster_minimal.tf"),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateId:           "name:tf-test-cluster-renamed",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"broker.private_key"},
				Config:                  GetClusterProvider() + GetFile("axual_cluster_minimal.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetClusterProvider() + GetFile("axual_cluster_minimal.tf"),
			},
		},
	})
}

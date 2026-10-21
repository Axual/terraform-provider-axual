package ClusterDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestClusterDataSource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutKafkaCluster(t, config)

	data, managed := "data.axual_cluster.tf_test_ds_cluster", "axual_cluster.tf_test_ds_cluster"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config:      GetClusterProvider() + GetFile("axual_cluster_data_source_missing.tf"),
				ExpectError: regexp.MustCompile(`no cluster "tf-test-cluster-that-does-not-exist" found`),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_cluster_data_source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(data, "id", managed, "id"),
					resource.TestCheckResourceAttr(data, "name", "tf-test-ds-cluster"),
					resource.TestCheckResourceAttr(data, "description", "Cluster the data source looks up"),
					resource.TestCheckResourceAttr(data, "location", "Utrecht"),
					resource.TestCheckResourceAttr(data, "provider_type", "apache-kafka"),
					resource.TestCheckResourceAttr(data, "shared", "false"),
					resource.TestCheckResourceAttr(data, "topic_pattern", "{tenant}-{instance}-{environment}-{topic}"),
					resource.TestCheckResourceAttr(data, "bootstrap_urls_by_authentication.MUTUAL_TLS", config.KafkaBootstrapServers),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetClusterProvider() + GetFile("axual_cluster_data_source.tf"),
			},
		},
	})
}

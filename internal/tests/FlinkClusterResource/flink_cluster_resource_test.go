package FlinkClusterResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestFlinkClusterResource(t *testing.T) {
	// The Ververica connection details come from test_config.yaml and are injected into the test
	// configuration as locals by GetProvider(), so the expected values are read from the same source.
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Test missing required attributes - should fail validation
			{
				Config:      GetProvider() + GetFile("axual_flink_cluster_missing_required.tf"),
				ExpectError: regexp.MustCompile(`is required`),
			},
			{
				Config: GetProvider() + GetFile("axual_flink_cluster_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_flink_cluster.tf_test_flink_cluster", "instance_id", "data.axual_instance.test_instance", "id"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "cluster_id", config.ClusterId),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "name", "tf-test-flink-cluster"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "description", "Axual's TF Test Flink Cluster"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "url", config.VervericaUrl),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "workspace", "defaultworkspace"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "namespace", "default"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "deployment_target", "default-target"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "api_token", config.VervericaApiToken),
					resource.TestCheckResourceAttrSet("axual_flink_cluster.tf_test_flink_cluster", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_flink_cluster_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "description", "Axual's TF Test Flink Cluster, updated"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "schema_registries.#", "1"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "schema_registries.0.type", "CONFLUENT"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "schema_registries.0.url", "https://schema-registry.example.com/apis/ccompat/v7"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_flink_cluster_no_description.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "name", "tf-test-flink-cluster-renamed"),
					resource.TestCheckNoResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "schema_registries.#"),
					resource.TestCheckNoResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "description"),
				),
			},
			{
				// A target Ververica does not know is refused by the API, which is what covers the
				// `deployment_target` update path: only one real target exists on the test Ververica.
				Config:      GetProvider() + GetFile("axual_flink_cluster_invalid_deployment_target.tf"),
				ExpectError: regexp.MustCompile(`deployment-target: tf-test-target-does-not-exist was not found`),
			},
			{
				// Unlike most resources, importing a Flink Cluster needs more than its own id: the uid is
				// only unique within the instance and cluster it lives on, so ImportState expects a
				// composite "instance_id/cluster_id/flink_cluster_id" identifier which ImportStateIdFunc
				// assembles from state.
				ResourceName: "axual_flink_cluster.tf_test_flink_cluster",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["axual_flink_cluster.tf_test_flink_cluster"]
					return rs.Primary.Attributes["instance_id"] + "/" + rs.Primary.Attributes["cluster_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				// the API does not return api_token on GET, so import cannot restore it.
				ImportStateVerifyIgnore: []string{"api_token"},
				ImportStateVerify:       true,
				Config:                  GetProvider() + GetFile("axual_flink_cluster_no_description.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_flink_cluster_no_description.tf"),
			},
		},
	})
}

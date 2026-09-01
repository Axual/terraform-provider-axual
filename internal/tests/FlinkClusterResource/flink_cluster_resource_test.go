package FlinkClusterResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestFlinkClusterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Test missing required attributes - should fail validation
			{
				Config:      GetProvider() + GetFile("internal/tests/FlinkClusterResource/axual_flink_cluster_missing_required.tf"),
				ExpectError: regexp.MustCompile(`is required`),
			},
			{
				Config: GetProvider() + GetFile("internal/tests/FlinkClusterResource/axual_flink_cluster_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_flink_cluster.tf_test_flink_cluster", "instance_id", "data.axual_instance.test_instance", "id"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "name", "tf-test-flink-cluster"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "description", "Axual's TF Test Flink Cluster"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "url", "https://vvp.example.internal/api"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "workspace", "defaultworkspace"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "namespace", "default"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "deployment_target", "default-target"),
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "api_token", "tf-test-initial-token"),
					resource.TestCheckResourceAttrSet("axual_flink_cluster.tf_test_flink_cluster", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile("internal/tests/FlinkClusterResource/axual_flink_cluster_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_flink_cluster.tf_test_flink_cluster", "description", "Axual's TF Test Flink Cluster, updated"),
				),
			},
			{
				ResourceName: "axual_flink_cluster.tf_test_flink_cluster",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["axual_flink_cluster.tf_test_flink_cluster"]
					return rs.Primary.Attributes["instance_id"] + "/" + rs.Primary.Attributes["cluster_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				// the API does not return api_token on GET, so import cannot restore it.
				ImportStateVerifyIgnore: []string{"api_token"},
				ImportStateVerify:       true,
				Config:                  GetProvider() + GetFile("internal/tests/FlinkClusterResource/axual_flink_cluster_updated.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("internal/tests/FlinkClusterResource/axual_flink_cluster_updated.tf"),
			},
		},
	})
}

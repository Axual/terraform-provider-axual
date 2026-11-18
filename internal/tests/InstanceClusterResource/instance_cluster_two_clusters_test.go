package InstanceClusterResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Two Clusters in one Instance, each with a schema registry: Terraform writes both in parallel, and
// Platform Manager 16.1.0 then answers one of them with an optimistic lock error, which is retried.
func TestInstanceClusterResourceTwoClusters(t *testing.T) {
	cfg, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutSchemaRegistry(t, cfg)

	files := GetClusterProvider() + GetFile("axual_instance_cluster_two_clusters.tf")
	check := func(step string) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(`axual_instance_cluster.tf_test_ic2["a"]`, "schema_registry.properties.tf.test.step", step),
			resource.TestCheckResourceAttr(`axual_instance_cluster.tf_test_ic2["b"]`, "schema_registry.properties.tf.test.step", step),
		)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config:          files,
				ConfigVariables: config.Variables{"step": config.StringVariable("create")},
				Check:           check("create"),
			},
			{
				// Both schema registries change in one apply.
				Config:          files,
				ConfigVariables: config.Variables{"step": config.StringVariable("update")},
				Check:           check("update"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy:         true,
				Config:          files,
				ConfigVariables: config.Variables{"step": config.StringVariable("update")},
			},
		},
	})
}

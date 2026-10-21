package InstanceResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const name = "axual_instance.tf_test_instance"

func TestInstanceResource(t *testing.T) {
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
				Config:      GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_invalid_short_name.tf"),
				ExpectError: regexp.MustCompile(`must be lower-case letters and digits`),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "name", "tf-test-instance"),
					resource.TestCheckResourceAttr(name, "short_name", "tftestinst"),
					resource.TestCheckResourceAttr(name, "description", "Axual's TF Test Instance"),
					resource.TestCheckResourceAttr(name, "clusters.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(name, "clusters.*", "axual_cluster.tf_test_instance_cluster_a", "id"),
					resource.TestCheckResourceAttr(name, "ca_certificates.#", "1"),
					// The defaults
					resource.TestCheckResourceAttr(name, "authentication_methods.#", "1"),
					resource.TestCheckTypeSetElemAttr(name, "authentication_methods.*", "MUTUAL_TLS"),
					resource.TestCheckResourceAttr(name, "granular_browse_permission", "false"),
					resource.TestCheckResourceAttr(name, "topic_owner_browse_permission", "false"),
					resource.TestCheckResourceAttr(name, "environment_mapping", "true"),
					resource.TestCheckResourceAttr(name, "self_approval", "true"),
					resource.TestCheckResourceAttr(name, "certificate_reuse_prevention", "NONE"),
					resource.TestCheckResourceAttr(name, "connect_enabled", "false"),
					resource.TestCheckResourceAttr(name, "flink_enabled", "false"),
					resource.TestCheckNoResourceAttr(name, "connect_urls"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-test-instance-renamed"),
					resource.TestCheckResourceAttr(name, "short_name", "tftestinst2"),
					resource.TestCheckResourceAttr(name, "description", "Axual's TF Test Instance, updated"),
					resource.TestCheckResourceAttr(name, "clusters.#", "2"),
					resource.TestCheckResourceAttr(name, "authentication_methods.#", "2"),
					resource.TestCheckTypeSetElemAttr(name, "authentication_methods.*", "SCRAM_SHA_512"),
					resource.TestCheckResourceAttr(name, "properties.replication.factor", "1"),
					resource.TestCheckResourceAttr(name, "granular_browse_permission", "true"),
					resource.TestCheckResourceAttr(name, "topic_owner_browse_permission", "true"),
					resource.TestCheckResourceAttr(name, "environment_mapping", "false"),
					resource.TestCheckResourceAttr(name, "self_approval", "false"),
					resource.TestCheckResourceAttr(name, "certificate_reuse_prevention", "TENANT"),
					resource.TestCheckResourceAttr(name, "flink_enabled", "true"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_minimal.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "clusters.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(name, "clusters.*", "axual_cluster.tf_test_instance_cluster_b", "id"),
					resource.TestCheckNoResourceAttr(name, "description"),
					resource.TestCheckResourceAttr(name, "authentication_methods.#", "1"),
					resource.TestCheckResourceAttr(name, "properties.%", "0"),
					resource.TestCheckResourceAttr(name, "granular_browse_permission", "false"),
					resource.TestCheckResourceAttr(name, "environment_mapping", "true"),
					resource.TestCheckResourceAttr(name, "self_approval", "true"),
					resource.TestCheckResourceAttr(name, "certificate_reuse_prevention", "NONE"),
					resource.TestCheckResourceAttr(name, "flink_enabled", "false"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Config:            GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_minimal.tf"),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     "name:tf-test-instance-renamed",
				ImportStateVerify: true,
				Config:            GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_minimal.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetClusterProvider() + GetFile("axual_instance_setup.tf", "axual_instance_minimal.tf"),
			},
		},
	})
}

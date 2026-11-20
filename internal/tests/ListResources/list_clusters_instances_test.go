package ListResources

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const queryClustersAndInstances = `
list "axual_cluster" "cluster" {
  provider         = axual
  include_resource = true
  config { name = "tf-query-cluster" }
}
list "axual_instance" "instance" {
  provider         = axual
  include_resource = true
  config { name = "tf-query-instance" }
}
`

func TestListClustersAndInstances(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutKafkaCluster(t, config)
	setup := GetClusterProvider() + GetFile("list_clusters_instances_setup.tf")

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: setup,
			},
			{
				Query:  true,
				Config: queryClustersAndInstances,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("axual_cluster.cluster", 1),
					querycheck.ExpectLength("axual_instance.instance", 1),
					querycheck.ExpectResourceKnownValues("axual_cluster.cluster",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-cluster")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("provider_type"), KnownValue: knownvalue.StringExact("apache-kafka")},
							{Path: tfjsonpath.New("broker").AtMapKey("bootstrap_servers"), KnownValue: knownvalue.StringExact(config.KafkaBootstrapServers)},
							{Path: tfjsonpath.New("ssl_auth_mode"), KnownValue: knownvalue.StringExact("fullChainCertificate")},
						}),
					querycheck.ExpectResourceKnownValues("axual_instance.instance",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-instance")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("short_name"), KnownValue: knownvalue.StringExact("tfqueryinst")},
							{Path: tfjsonpath.New("granular_browse_permission"), KnownValue: knownvalue.Bool(true)},
						}),
				},
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  setup,
			},
		},
	})
}

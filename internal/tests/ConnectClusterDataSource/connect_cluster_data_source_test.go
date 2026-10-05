package ConnectClusterDataSource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestConnectClusterDataSource looks up two Kafka Connect clusters that must already be registered
// on the local stack by an administrator (there is no axual_connect_cluster resource): one
// MTLS-authenticated, one SASL_SCRAM-authenticated, exercising both lookup-by-id and
// lookup-by-name, plus the not-found and missing-attribute error paths.
func TestConnectClusterDataSource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutConnectCluster(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetConnectProvider() + GetFile("axual_connect_cluster_by_id.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_id", "id", config.ConnectClusterMtlsId),
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_id", "name", config.ConnectClusterMtlsName),
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_id", "auth_method", "MTLS"),
					resource.TestCheckResourceAttrSet("data.axual_connect_cluster.by_id", "connect_url"),
					resource.TestCheckResourceAttrSet("data.axual_connect_cluster.by_id", "owner_group_id"),
				),
			},
			{
				// Looked up by name instead of id: reads every page of the list (the API has no
				// server-side name filter), then a single GET for the full detail.
				Config: GetConnectProvider() + GetFile("axual_connect_cluster_by_name.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_name", "id", config.ConnectClusterSaslId),
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_name", "name", config.ConnectClusterSaslName),
					resource.TestCheckResourceAttr("data.axual_connect_cluster.by_name", "auth_method", "SASL_SCRAM"),
				),
			},
			{
				Config:      GetConnectProvider() + GetFile("axual_connect_cluster_not_found.tf"),
				ExpectError: regexp.MustCompile("Resource Not Found"),
			},
			{
				Config:      GetConnectProvider() + GetFile("axual_connect_cluster_without_id_name.tf"),
				ExpectError: regexp.MustCompile("Missing Attribute Configuration"),
			},
			{
				Destroy: true,
				Config:  GetConnectProvider() + GetFile("axual_connect_cluster_by_id.tf"),
			},
		},
	})
}

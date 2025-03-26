package SchemaVersionDataSource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaVersionWithOwnerDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_with_owner.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "full_name", "io.axual.qa.general.GitOpsTest1"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.test_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.test_v1_imported", "id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.test_v1_imported", "owners"),
					CheckBodyMatchesFile("data.axual_schema_version.test_v1_imported", "body", "avro-schemas/gitops_test_v1.avsc"),
				),
			},
		},
	})
}

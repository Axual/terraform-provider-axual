package SchemaVersionDataSource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Regression test: the data source used to only ever match the first schema version in the
// list Platform Manager returns. Looking up any version other than that one (in practice, any
// version older than the latest) failed with "Schema version matching the name you requested
// was not found", even though the version genuinely existed.
func TestSchemaVersionDataSource_findsOlderVersionAfterNewerOneExists(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_multiple_versions.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "description", "Multi-version gitops test schema"),
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "full_name", "io.axual.qa.general.GitOpsTestMultiVersion"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.multi_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.multi_v1_imported", "id"),
					CheckBodyMatchesFile("data.axual_schema_version.multi_v1_imported", "body", "avro-schemas/gitops_test_multi_version_v1.avsc"),
				),
			},
		},
	})
}

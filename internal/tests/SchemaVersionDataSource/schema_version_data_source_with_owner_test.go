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
					resource.TestCheckResourceAttr("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "full_name", "io.axual.qa.general.GitOpsTest1"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "id"),
					resource.TestCheckResourceAttrPair("data.axual_schema_version.axual_gitops_test_schema_version_with_owner", "owners.id", "data.axual_group.group", "5a32dae2dbee4249a6cef8ce8759bd94"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version_with_owner", "body", "avro-schemas/gitops_test_v1.avsc"),
				),
			},
		},
	})
}

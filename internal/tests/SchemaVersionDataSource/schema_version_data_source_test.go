package SchemaVersionDataSource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaVersionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.test_v1_imported", "full_name", "io.axual.qa.general.GitOpsTest1"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.test_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.test_v1_imported", "id"),
					CheckBodyMatchesFile("data.axual_schema_version.test_v1_imported", "body", "avro-schemas/gitops_test_v1.avsc"),

					resource.TestCheckResourceAttr("data.axual_schema_version.protobuf_v1_imported", "description", "AddressBook schema"),
					resource.TestCheckResourceAttr("data.axual_schema_version.protobuf_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.protobuf_v1_imported", "full_name", "AddressBook"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.protobuf_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.protobuf_v1_imported", "id"),
					CheckBodyMatchesFile("data.axual_schema_version.protobuf_v1_imported", "body", "protobuf-schemas/tf-protobuf-test1.proto"),

					resource.TestCheckResourceAttr("data.axual_schema_version.jsonschema_v1_imported", "description", "Person schema"),
					resource.TestCheckResourceAttr("data.axual_schema_version.jsonschema_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.jsonschema_v1_imported", "full_name", "Person"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.jsonschema_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.jsonschema_v1_imported", "id"),
					CheckBodyMatchesFile("data.axual_schema_version.jsonschema_v1_imported", "body", "json-schemas/tf-json-schema-test1.json"),
				),
			},
		},
	})
}

// TestSchemaVersionDataSourceMultipleVersions is a regression test: the data source used to only
// ever match the first schema version in the list Platform Manager returns. Looking up any
// version other than that one (in practice, any version older than the latest) failed with
// "Schema version matching the name you requested was not found", even though the version
// genuinely existed. description is checked as "version 2" here, not "version 1": it belongs to
// the schema, not to the specific version looked up, so it reflects whichever version last set it.
//
// ExpectNonEmptyPlan is required and expected: description lives on the schema, shared by every
// axual_schema_version for it, so multi_v2 setting a different description is real drift for
// multi_v1's own state on refresh (it still declares its own, now-stale value) - not a test bug.
// A real user hits the same thing giving two versions of one schema different descriptions.
func TestSchemaVersionDataSourceMultipleVersions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:             GetProvider() + GetFile("axual_schema_version_multiple_versions.tf"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "version", "1.0.0"),
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "description", "Multi-version gitops test schema, version 2"),
					resource.TestCheckResourceAttr("data.axual_schema_version.multi_v1_imported", "full_name", "io.axual.qa.general.GitOpsTestMultiVersion"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.multi_v1_imported", "schema_id"),
					resource.TestCheckResourceAttrSet("data.axual_schema_version.multi_v1_imported", "id"),
					CheckBodyMatchesFile("data.axual_schema_version.multi_v1_imported", "body", "avro-schemas/gitops_test_multi_version_v1.avsc"),
				),
			},
		},
	})
}

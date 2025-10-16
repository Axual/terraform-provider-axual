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

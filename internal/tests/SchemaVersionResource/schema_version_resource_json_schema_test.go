package SchemaVersionResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaVersionJSONSchemaResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_json_schema_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "description", "Person schema"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "type", "JSON_SCHEMA"),
					CheckBodyMatchesFile("axual_schema_version.test_json_v1", "body", "json-schemas/tf-json-schema-test1.json"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_schema_version_json_multiple_versions_for_same_schema.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "description", "Person schema"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "full_name", "Person"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "type", "JSON_SCHEMA"),
					CheckBodyMatchesFile("axual_schema_version.test_json_v1", "body", "json-schemas/tf-json-schema-test1.json"),

					resource.TestCheckResourceAttr("axual_schema_version.test_json_v2", "version", "2.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v2", "description", "Person schema"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v2", "full_name", "Person"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v2", "type", "JSON_SCHEMA"),
					CheckBodyMatchesFile("axual_schema_version.test_json_v2", "body", "json-schemas/tf-json-schema-test2.json"),

					resource.TestCheckResourceAttr("axual_schema_version.test_json_v3", "version", "3.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v3", "description", "Person schema"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v3", "full_name", "Person"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v3", "type", "JSON_SCHEMA"),
					CheckBodyMatchesFile("axual_schema_version.test_json_v3", "body", "json-schemas/tf-json-schema-test3.json"),

					// to verify that the both schema versions belong to the same schema
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "description",
						"axual_schema_version.test_json_v2", "description",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "description",
						"axual_schema_version.test_json_v3", "description",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "full_name",
						"axual_schema_version.test_json_v2", "full_name",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "full_name",
						"axual_schema_version.test_json_v3", "full_name",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "schema_id",
						"axual_schema_version.test_json_v2", "schema_id",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_json_v1", "schema_id",
						"axual_schema_version.test_json_v3", "schema_id",
					),
				),
			},
			{
				ResourceName:      "axual_schema_version.test_json_v1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_json_multiple_versions_for_same_schema.tf"),
			},
		},
	})
}

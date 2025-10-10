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
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "description", "Gitops test JSON Schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.test_json_v1", "type", "JSON_SCHEMA"),
					CheckBodyMatchesFile("axual_schema_version.test_json_v1", "body", "json-schemas/tf-json-schema-test1.json"),
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
				Config:  GetProvider() + GetFile("axual_schema_version_json_schema_initial.tf"),
			},
		},
	})
}

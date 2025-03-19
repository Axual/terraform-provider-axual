package SchemaVersionResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaVersionProtobufResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_protobuf_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "description", "Gitops test protobuf schema version"),
					CheckBodyMatchesFile("axual_schema_version.test_v1", "body", "avro-schemas/tf-protobuf-test1.proto"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_protobuf_initial.tf"),
			},
		},
	})
}

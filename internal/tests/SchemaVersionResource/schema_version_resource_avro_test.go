package SchemaVersionResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSchemaVersionAvroResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/gitops_test_v1.avsc"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version1"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_v2_replaced.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version2"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "2.0.0"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/gitops_test_v2_backwards_compatible.avsc"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_v3_replaced.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version2"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "3.0.0"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/gitops_test_v3_forwards_compatible.avsc"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_avro_v3_replaced.tf"),
			},
		},
	})
}
func TestSchemaVersionAvroWithOwnersResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_with_owner.tf"),
				Check: resource.ComposeTestCheckFunc(

					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version_with_owner", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version_with_owner", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version_with_owner", "full_name", "io.axual.qa.general.GitOpsTest1"),
					resource.TestCheckResourceAttrSet("axual_schema_version.axual_gitops_test_schema_version_with_owner", "schema_id"),
					resource.TestCheckResourceAttrSet("axual_schema_version.axual_gitops_test_schema_version_with_owner", "id"),
					resource.TestCheckResourceAttrPair("axual_schema_version.axual_gitops_test_schema_version_with_owner", "owners.id", "data.axual_group.group", "5a32dae2dbee4249a6cef8ce8759bd94"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version_with_owner", "body", "avro-schemas/gitops_test_v1.avsc"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_avro_v3_replaced.tf"),
			},
		},
	})
}

//TODO when Terraform Provider supports Protobuf schemas
//func TestSchemaVersionProtobufResource(t *testing.T) {
//	resource.Test(t, resource.TestCase{
//		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
//		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
//
//		Steps: []resource.TestStep{
//			{
//				Config: GetProvider() + GetFile("axual_schema_version_protobuf_initial.tf"),
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "1.0.0"),
//					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test protobuf schema version"),
//					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/tf-protobuf-test1.proto"),
//				),
//			},
//			//{
//			//	Config: GetProvider() + GetFile("axual_schema_version_avro_updated.tf"),
//			//	Check: resource.ComposeTestCheckFunc(
//			//		resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version1"),
//			//	),
//			//},
//			//{
//			//	Config: GetProvider() + GetFile("axual_schema_version_avro_replaced.tf"),
//			//	Check: resource.ComposeTestCheckFunc(
//			//		resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version2"),
//			//		resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "2.0.0"),
//			//		CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/gitops_test_v2.avsc"),
//			//	),
//			//},
//			{
//				// To ensure cleanup if one of the test cases had an error
//				Destroy: true,
//				Config:  GetProvider() + GetFile("axual_schema_version_avro_replaced.tf"),
//			},
//		},
//	})
//}

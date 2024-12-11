package SchemaVersionResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
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
				Config:      GetProvider() + GetFile("axual_schema_version_avro_desc_updated.tf"),
				ExpectError: regexp.MustCompile(`(?s)API does not allow update of schema version\. Please create another version of\s+the schema`),
			},
			{
				Config:      GetProvider() + GetFile("axual_schema_version_avro_v2_replaced.tf"),
				ExpectError: regexp.MustCompile(`(?s)API does not allow update of schema version\. Please create another version of\s+the schema`),
			},
			{
				Config:      GetProvider() + GetFile("axual_schema_version_avro_v3_replaced.tf"),
				ExpectError: regexp.MustCompile(`(?s)API does not allow update of schema version\. Please create another version of\s+the schema`),
			},
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_multiple_versions_for_same_schema.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "full_name", "io.axual.qa.general.GitOpsTest1"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version1", "body", "avro-schemas/gitops_test_v1.avsc"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version2", "version", "2.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version1", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version2", "full_name", "io.axual.qa.general.GitOpsTest1"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version2", "body", "avro-schemas/gitops_test_v2_backwards_compatible.avsc"),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.axual_gitops_test_schema_version1", "description",
						"axual_schema_version.axual_gitops_test_schema_version2", "description",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.axual_gitops_test_schema_version1", "full_name",
						"axual_schema_version.axual_gitops_test_schema_version2", "full_name",
					),
					// to verify boh schema versions belongs to the same schema
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.axual_gitops_test_schema_version1", "schema_id",
						"axual_schema_version.axual_gitops_test_schema_version2", "schema_id",
					),
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
					resource.TestCheckResourceAttr("axual_schema_version.axual_gitops_test_schema_version_with_owner", "full_name", "io.axual.general.GitOpsTest2"),
					resource.TestCheckResourceAttrSet("axual_schema_version.axual_gitops_test_schema_version_with_owner", "schema_id"),
					resource.TestCheckResourceAttrSet("axual_schema_version.axual_gitops_test_schema_version_with_owner", "id"),
					resource.TestCheckResourceAttrSet("axual_schema_version.axual_gitops_test_schema_version_with_owner", "owners"),
					CheckBodyMatchesFile("axual_schema_version.axual_gitops_test_schema_version_with_owner", "body", "avro-schemas/gitops_test_v2.avsc"),
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
//			//	Config: GetProvider() + GetFile("axual_schema_version_avro_desc_updated.tf"),
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

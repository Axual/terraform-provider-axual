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
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "description", "Gitops test schema version"),
					CheckBodyMatchesFile("axual_schema_version.test_v1", "body", "avro-schemas/gitops_test_v1.avsc"),
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
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "full_name", "io.axual.qa.general.GitOpsTest1"),
					CheckBodyMatchesFile("axual_schema_version.test_v1", "body", "avro-schemas/gitops_test_v1.avsc"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v2", "version", "2.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v1", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v2", "full_name", "io.axual.qa.general.GitOpsTest1"),
					CheckBodyMatchesFile("axual_schema_version.test_v2", "body", "avro-schemas/gitops_test_v2_backwards_compatible.avsc"),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_v1", "description",
						"axual_schema_version.test_v2", "description",
					),
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_v1", "full_name",
						"axual_schema_version.test_v2", "full_name",
					),
					// to verify that the both schema versions belong to the same schema
					resource.TestCheckResourceAttrPair(
						"axual_schema_version.test_v1", "schema_id",
						"axual_schema_version.test_v2", "schema_id",
					),
				),
			},
			{
				ResourceName:      "axual_schema_version.test_v1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_avro_multiple_versions_for_same_schema.tf"),
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

					resource.TestCheckResourceAttr("axual_schema_version.test_v2_with_owner", "description", "Gitops test schema version"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v2_with_owner", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_v2_with_owner", "full_name", "io.axual.general.GitOpsTest2"),
					resource.TestCheckResourceAttrSet("axual_schema_version.test_v2_with_owner", "schema_id"),
					resource.TestCheckResourceAttrSet("axual_schema_version.test_v2_with_owner", "id"),
					resource.TestCheckResourceAttrSet("axual_schema_version.test_v2_with_owner", "owners"),
					CheckBodyMatchesFile("axual_schema_version.test_v2_with_owner", "body", "avro-schemas/gitops_test_v2.avsc"),
				),
			},
			{
				ResourceName:      "axual_schema_version.test_v2_with_owner",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_with_owner.tf"),
			},
		},
	})
}

func TestSchemaVersionAvroWithExplicitTypeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_schema_version_avro_with_type.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_schema_version.test_avro_explicit_type_v1", "version", "1.0.0"),
					resource.TestCheckResourceAttr("axual_schema_version.test_avro_explicit_type_v1", "description", "Gitops test schema version with explicit AVRO type"),
					resource.TestCheckResourceAttr("axual_schema_version.test_avro_explicit_type_v1", "type", "AVRO"),
					CheckBodyMatchesFile("axual_schema_version.test_avro_explicit_type_v1", "body", "avro-schemas/gitops_test_v1.avsc"),
				),
			},
			{
				ResourceName:      "axual_schema_version.test_avro_explicit_type_v1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_schema_version_avro_with_type.tf"),
			},
		},
	})
}

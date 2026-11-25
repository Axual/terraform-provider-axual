package KsmlRuntimeResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const ksmlRuntime = "axual_ksml_runtime.tf_test_ksml_runtime"

func TestKsmlRuntimeResource(t *testing.T) {
	// The cluster and the KSML runtime URL come from test_config.yaml and are injected into the test
	// configuration as locals by GetKsmlProvider(), so the expected values are read from the same source.
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutKsmlRuntime(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Test missing required attributes - should fail validation
			{
				Config:      GetKsmlProvider() + GetFile("axual_ksml_runtime_missing_required.tf"),
				ExpectError: regexp.MustCompile(`is required`),
			},
			{
				Config:      GetKsmlProvider() + GetFile("axual_ksml_runtime_missing_owner.tf"),
				ExpectError: regexp.MustCompile(`"owners" is required`),
			},
			{
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(ksmlRuntime, "instance_id", "data.axual_instance.test_instance", "id"),
					resource.TestCheckResourceAttr(ksmlRuntime, "cluster_id", config.ClusterId),
					resource.TestCheckResourceAttr(ksmlRuntime, "name", "tf-test-ksml-runtime"),
					resource.TestCheckResourceAttr(ksmlRuntime, "description", "Axual's TF Test KSML runtime"),
					resource.TestCheckResourceAttr(ksmlRuntime, "url", config.KsmlRuntimeUrl),
					resource.TestCheckResourceAttrSet(ksmlRuntime, "id"),
					resource.TestCheckResourceAttrPair(ksmlRuntime, "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(ksmlRuntime, "authorized_groups.*", "data.axual_group.test_group", "id"),
					resource.TestCheckNoResourceAttr(ksmlRuntime, "schema_registries.#"),
				),
			},
			{
				// Leaving authorized_groups out keeps the Groups the KSML runtime has.
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_authorized_groups_omitted.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(ksmlRuntime, "authorized_groups.*", "data.axual_group.test_group", "id"),
				),
			},
			{
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_updated.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ksmlRuntime, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ksmlRuntime, "description", "Axual's TF Test KSML runtime, updated"),
					resource.TestCheckResourceAttr(ksmlRuntime, "url", "https://tf-test-ksml-runtime-updated.example.com"),
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "0"),
					resource.TestCheckResourceAttr(ksmlRuntime, "schema_registries.#", "1"),
					resource.TestCheckResourceAttr(ksmlRuntime, "schema_registries.0.type", "CONFLUENT"),
					resource.TestCheckResourceAttr(ksmlRuntime, "schema_registries.0.url", "https://schema-registry.example.com/apis/ccompat/v7"),
				),
			},
			{
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_no_description.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ksmlRuntime, "name", "tf-test-ksml-runtime"),
					resource.TestCheckResourceAttr(ksmlRuntime, "url", config.KsmlRuntimeUrl),
					resource.TestCheckNoResourceAttr(ksmlRuntime, "schema_registries.#"),
					resource.TestCheckNoResourceAttr(ksmlRuntime, "description"),
					// authorized_groups was [] and is left out, so it stays empty.
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "0"),
					resource.TestCheckResourceAttrPair(ksmlRuntime, "owners", "data.axual_group.test_group", "id"),
				),
			},
			{
				// Platform Manager does not rename a KSML runtime, so a changed name replaces it.
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_renamed.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ksmlRuntime, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ksmlRuntime, "name", "tf-test-ksml-runtime-renamed"),
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "1"),
				),
			},
			{
				// ImportState expects the Instance-Cluster the KSML runtime is registered on and the KSML runtime, as
				// a composite "instance_id/cluster_id/ksml_runtime_id" identifier, which ImportStateIdFunc
				// assembles from state.
				ResourceName: ksmlRuntime,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[ksmlRuntime]
					return rs.Primary.Attributes["instance_id"] + "/" + rs.Primary.Attributes["cluster_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerify: true,
				Config:            GetKsmlProvider() + GetFile("axual_ksml_runtime_renamed.tf"),
			},
			{
				// A replaced KSML runtime gets only the Groups in the configuration: with authorized_groups
				// left out it has none, whatever the one it replaces had.
				Config: GetKsmlProvider() + GetFile("axual_ksml_runtime_replaced_authorized_groups_omitted.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(ksmlRuntime, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ksmlRuntime, "name", "tf-test-ksml-runtime-replaced"),
					resource.TestCheckResourceAttr(ksmlRuntime, "authorized_groups.#", "0"),
				),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetKsmlProvider() + GetFile("axual_ksml_runtime_replaced_authorized_groups_omitted.tf"),
			},
		},
	})
}

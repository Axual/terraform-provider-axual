package InstanceClusterResource

import (
	"fmt"
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const name = "axual_instance_cluster.tf_test_ic"

func TestInstanceClusterResource(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	SkipWithoutSchemaRegistry(t, config)

	setup := "axual_instance_cluster_setup.tf"
	var publicId string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config:      GetClusterProvider() + GetFile(setup, "axual_instance_cluster_username_without_password.tf"),
				ExpectError: regexp.MustCompile(`password`),
			},
			{
				Config:      GetClusterProvider() + GetFile(setup, "axual_instance_cluster_api_type_mismatch.tf"),
				ExpectError: regexp.MustCompile(`api does not match type`),
			},
			{
				Config:      GetClusterProvider() + GetFile(setup, "axual_instance_cluster_endpoints_and_listeners.tf"),
				ExpectError: regexp.MustCompile(`cannot be specified when`),
			},
			{
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(name, "instance", "axual_instance.tf_test_ic_instance", "id"),
					resource.TestCheckResourceAttrPair(name, "cluster", "axual_cluster.tf_test_ic_cluster", "id"),
					resource.TestCheckResourceAttr(name, "schema_registry.type", "APICURIO"),
					resource.TestCheckResourceAttr(name, "schema_registry.url", config.SchemaRegistryUrl),
					resource.TestCheckResourceAttr(name, "schema_registry.authentication", "Basic-Auth"),
					resource.TestCheckResourceAttr(name, "schema_registry.username", config.SchemaRegistryUsername),
					resource.TestCheckResourceAttr(name, "schema_registry.oauth_enabled", "false"),
					resource.TestCheckNoResourceAttr(name, "schema_registry.version"),
				),
			},
			{
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_confluent.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.type", "CONFLUENT"),
					resource.TestCheckResourceAttr(name, "schema_registry.authentication", "No-Auth"),
					resource.TestCheckResourceAttr(name, "schema_registry.version", "7.0"),
					resource.TestCheckNoResourceAttr(name, "schema_registry.username"),
				),
			},
			// The steps with api and client_endpoints need Platform Manager 16.1.0. Before that, the plan
			// fails, see the unit tests.
			{
				// Platform Manager derives api from version, and the endpoints from listeners: APICURIO_V2
				// for this Apicurio 2.x registry before 17.0.0 (AXPD-12769), APICURIO_V3 since. Master
				// builds still report 16.1.0 (no 17.0.0 version bump yet), so accept both; the legacy
				// endpoint must have the same api.
				SkipFunc: SkipStepWithoutSchemaRegistryApi,
				Config:   GetClusterProvider() + GetFile(setup, "axual_instance_cluster_derived.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(name, "schema_registry.api", regexp.MustCompile(`^APICURIO_V[23]$`)),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.%", "1"),
					resource.TestCheckResourceAttrPair(name, "schema_registry.client_endpoints.legacy.api", name, "schema_registry.api"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.legacy.url", config.SchemaRegistryUrl),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.legacy.used_by", "ALL_APPS"),
				),
			},
			{
				SkipFunc: SkipStepWithoutSchemaRegistryApi,
				Config:   GetClusterProvider() + GetFile(setup, "axual_instance_cluster_endpoints.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.api", "APICURIO_V3"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.%", "2"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.internal.used_by", "MANAGED_APPS"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.public.used_by", "ALL_APPS"),
					resource.TestCheckResourceAttr(name, "schema_registry.listeners.%", "2"),
					resource.TestCheckResourceAttr(name, "schema_registry.listeners.public", "https://schema-registry.example.com/apis/registry/v3"),
					checkEndpointId("public", &publicId),
				),
			},
			{
				// public changes in place and keeps its id in Platform Manager; internal is removed.
				SkipFunc: SkipStepWithoutSchemaRegistryApi,
				Config:   GetClusterProvider() + GetFile(setup, "axual_instance_cluster_endpoints_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.%", "1"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.public.api", "APICURIO_V2"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.public.url", "https://schema-registry.example.com/apis/registry/v2"),
					resource.TestCheckResourceAttr(name, "schema_registry.client_endpoints.public.used_by", "MANAGED_APPS"),
					resource.TestCheckResourceAttr(name, "schema_registry.listeners.%", "1"),
					checkEndpointId("public", &publicId),
				),
			},
			{
				SkipFunc:                SkipStepWithoutSchemaRegistryApi,
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"schema_registry.password"},
				Config:                  GetClusterProvider() + GetFile(setup, "axual_instance_cluster_endpoints_updated.tf"),
			},
			{
				// An AVRO topic on an environment of the Instance registers its schemas in the
				// registry of the Instance-Cluster.
				Config: GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "schema_registry.type", "APICURIO"),
					resource.TestCheckResourceAttr(name, "schema_registry.version", "3.0"),
					resource.TestCheckResourceAttrSet("axual_topic_config.tf_test_ic_topic_config", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_config.tf_test_ic_topic_config", "value_schema_version", "axual_schema_version.tf_test_ic_value", "id"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// Platform Manager never returns the password, so import cannot restore it.
				ImportStateVerifyIgnore: []string{"schema_registry.password"},
				Config:                  GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetClusterProvider() + GetFile(setup, "axual_instance_cluster_with_topic.tf"),
			},
		},
	})
}

// checkEndpointId reads the Platform Manager id of a client endpoint. The first call remembers it in
// id; a later call fails when the id changed, so the endpoint was made new instead of changed in place.
func checkEndpointId(endpoint string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("no resource %q in state", name)
		}
		client, err := ApiClient()
		if err != nil {
			return err
		}
		registry, err := client.GetInstanceClusterSchemaRegistry(rs.Primary.Attributes["instance"], rs.Primary.Attributes["cluster"])
		if err != nil {
			return err
		}
		if registry.ClientEndpoints != nil {
			for _, e := range *registry.ClientEndpoints {
				if e.Name != endpoint {
					continue
				}
				if *id == "" {
					*id = e.Id
				} else if e.Id != *id {
					return fmt.Errorf("endpoint %q has id %s, expected %s: it was made new instead of changed in place", endpoint, e.Id, *id)
				}
				return nil
			}
		}
		return fmt.Errorf("Platform Manager has no client endpoint %q", endpoint)
	}
}

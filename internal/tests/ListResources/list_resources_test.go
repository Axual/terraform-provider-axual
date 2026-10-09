package ListResources

import (
	"fmt"
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// The list blocks below select only the resources created by list_resources_setup.tf, by
// environment, name, or schema.
const queryAll = `
list "axual_environment" "env" {
  provider = axual
  config { name = "tf-query-env" }
}
list "axual_topic" "topic" {
  provider         = axual
  include_resource = true
  config { name = "tf-query-topic" }
}
list "axual_topic_config" "topic_config" {
  provider         = axual
  include_resource = true
  config { environment = "tfqueryenv" }
}
list "axual_application" "apps" {
  provider         = axual
  include_resource = true
  config { short_name = "tfquery" }
}
list "axual_application_credential" "credential" {
  provider = axual
  config { environment = "tfqueryenv" }
}
list "axual_application_access_grant" "grants" {
  provider = axual
  config { environment = "tfqueryenv" }
}
list "axual_application_access_grant" "producer_grants" {
  provider = axual
  config {
    environment = "tfqueryenv"
    access_type = "PRODUCER"
  }
}
list "axual_application_access_grant_approval" "approval" {
  provider = axual
  config { environment = "tfqueryenv" }
}
list "axual_application_access_grant_rejection" "rejection" {
  provider = axual
  config { environment = "tfqueryenv" }
}
list "axual_application_deployment" "deployment" {
  provider         = axual
  include_resource = true
  config { environment = "tfqueryenv" }
}
list "axual_application_deployment_state" "deployment_state" {
  provider = axual
  config { application = "tfqueryconnector" }
}
list "axual_schema_version" "schema" {
  provider = axual
  config { schema = "io.axual.tfquery.Person" }
}
`

// queryByOwnerName checks that a group can be given by its name.
const queryByOwnerName = `
list "axual_topic" "owned" {
  provider = axual
  config {
    name   = "tf-query-topic"
    owners = "%s"
  }
}
`

func TestListResources(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	setup := GetProvider() + GetFile("list_resources_setup.tf")

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: setup,
			},
			{
				Query:  true,
				Config: queryAll,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("axual_environment.env", 1),
					querycheck.ExpectLength("axual_topic.topic", 1),
					querycheck.ExpectLength("axual_topic_config.topic_config", 1),
					querycheck.ExpectLength("axual_application.apps", 2),
					querycheck.ExpectLength("axual_application_credential.credential", 1),
					querycheck.ExpectLength("axual_application_access_grant.grants", 2),
					querycheck.ExpectLength("axual_application_access_grant.producer_grants", 1),
					querycheck.ExpectLength("axual_application_access_grant_approval.approval", 1),
					querycheck.ExpectLength("axual_application_access_grant_rejection.rejection", 1),
					querycheck.ExpectLength("axual_application_deployment.deployment", 1),
					querycheck.ExpectLength("axual_application_deployment_state.deployment_state", 1),
					querycheck.ExpectLength("axual_schema_version.schema", 1),

					querycheck.ExpectResourceDisplayName("axual_topic_config.topic_config",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-topic on tfqueryenv")),
						knownvalue.StringExact("tf-query-topic on tfqueryenv")),
					querycheck.ExpectResourceDisplayName("axual_application_deployment.deployment",
						queryfilter.ByDisplayName(knownvalue.StringExact("tfqueryconnector on tfqueryenv")),
						knownvalue.StringExact("tfqueryconnector on tfqueryenv")),

					querycheck.ExpectResourceKnownValues("axual_topic.topic",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-topic")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("key_type"), KnownValue: knownvalue.StringExact("String")},
							{Path: tfjsonpath.New("retention_policy"), KnownValue: knownvalue.StringExact("delete")},
							{Path: tfjsonpath.New("description"), KnownValue: knownvalue.StringExact("Topic for the terraform query tests")},
						}),
					querycheck.ExpectResourceKnownValues("axual_topic_config.topic_config",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-topic on tfqueryenv")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("partitions"), KnownValue: knownvalue.Int64Exact(1)},
							{Path: tfjsonpath.New("retention_time"), KnownValue: knownvalue.Int64Exact(864000)},
						}),
					querycheck.ExpectResourceKnownValues("axual_application.apps",
						queryfilter.ByDisplayName(knownvalue.StringExact("tf-query-custom")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("application_type"), KnownValue: knownvalue.StringExact("Custom")},
							{Path: tfjsonpath.New("short_name"), KnownValue: knownvalue.StringExact("tfquerycustom")},
						}),
					querycheck.ExpectResourceKnownValues("axual_application_deployment.deployment",
						queryfilter.ByDisplayName(knownvalue.StringExact("tfqueryconnector on tfqueryenv")),
						[]querycheck.KnownValueCheck{
							{Path: tfjsonpath.New("autostart"), KnownValue: knownvalue.Bool(true)},
							{Path: tfjsonpath.New("configs").AtMapKey("logger.name"), KnownValue: knownvalue.StringExact("tfquerylogger")},
						}),
				},
			},
			{
				Query:  true,
				Config: fmt.Sprintf(queryByOwnerName, config.GroupName),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("axual_topic.owned", 1),
				},
			},
			{
				Query: true,
				Config: `
list "axual_topic_config" "missing" {
  provider = axual
  config { environment = "tf-query-no-such-env" }
}
`,
				ExpectError: regexp.MustCompile(`no environment found with ID or name "tf-query-no-such-env"`),
			},
			{
				// Import with an `identity` instead of an ID, for resources with an "id" identity and one
				// with a different identity attribute.
				Config:          setup,
				ResourceName:    "axual_topic.tf-query-topic",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config:          setup,
				ResourceName:    "axual_application.tf-query-custom",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config:          setup,
				ResourceName:    "axual_topic_config.tf-query-topic-config",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config:          setup,
				ResourceName:    "axual_application_access_grant_approval.tf-query-producer-approval",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Destroy: true,
				Config:  setup,
			},
		},
	})
}

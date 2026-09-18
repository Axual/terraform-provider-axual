package GroupResource

import (
	"fmt"
	"strings"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_group_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-integrations", "name", "testgroup9999"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "email_address", "test.user@axual.com"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "phone_number", "+6112356789"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "members.#", "1"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "managers.#", "1"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_group_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-integrations", "name", "updatedgroup9999"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "email_address", "updated.user@axual.com"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "phone_number", "+61123456789"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "members.#", "0"),
					resource.TestCheckResourceAttr("axual_group.team-integrations", "managers.#", "0"),
				),
			},
			{
				ResourceName:      "axual_group.team-integrations",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_group_updated.tf"),
			},
		},
	})
}

// The unit tests in axual-webclient/groups_test.go cover which URI is built for which member.
// What they cannot cover is Platform Manager actually accepting it, which is this test's whole
// point: a mixed member list written and read back against a live API.
func TestGroupResourceServiceAccountMember(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	if config.ServiceAccountUid == "" || strings.HasPrefix(config.ServiceAccountUid, "YOUR_") {
		t.Skip("serviceAccountUid is not set in test_config.yaml, so no service account can be added to a group")
	}
	locals := fmt.Sprintf("locals {\n  service_account_uid = %q\n}\n", config.ServiceAccountUid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + locals + GetFile("axual_group_service_account_member.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-service-accounts", "name", "testgroupsa9999"),
					resource.TestCheckResourceAttr("axual_group.team-service-accounts", "members.#", "2"),
					resource.TestCheckTypeSetElemAttr("axual_group.team-service-accounts", "members.*", config.ServiceAccountUid),
				),
			},
			{
				ResourceName:      "axual_group.team-service-accounts",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// A manager uid is looked up the same way a member uid is (GroupMemberURI), since a manager is a
// member with an extra role, not a group - this exercises that same lookup for the managers list.
func TestGroupResourceServiceAccountManager(t *testing.T) {
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	if config.ServiceAccountUid == "" || strings.HasPrefix(config.ServiceAccountUid, "YOUR_") {
		t.Skip("serviceAccountUid is not set in test_config.yaml, so no service account can be added to a group")
	}
	locals := fmt.Sprintf("locals {\n  service_account_uid = %q\n}\n", config.ServiceAccountUid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + locals + GetFile("axual_group_service_account_manager.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_group.team-service-account-manager", "name", "testgroupsamgr9999"),
					resource.TestCheckResourceAttr("axual_group.team-service-account-manager", "managers.#", "2"),
					resource.TestCheckTypeSetElemAttr("axual_group.team-service-account-manager", "managers.*", config.ServiceAccountUid),
				),
			},
			{
				ResourceName:      "axual_group.team-service-account-manager",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

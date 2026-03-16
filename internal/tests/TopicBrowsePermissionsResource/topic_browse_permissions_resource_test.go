package TopicBrowsePermissionsResource

import (
	"fmt"
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestTopicBrowsePermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create the initial resource with 1 user and 1 group
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.0", "data.axual_user.ben", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.0", "axual_group.team-group", "id"),
				),
			},
			// Step 2: Import test
			{
				ResourceName:                         "axual_topic_browse_permissions.tf-test-topic-browse-permissions",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "topic_config",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["axual_topic_browse_permissions.tf-test-topic-browse-permissions"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["topic_config"], nil
				},
			},
			// Step 3: Update by removing the users (only the groups field remains, users field is removed)
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_updated_remove_user.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.#", "0"),
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.0", "axual_group.team-group", "id"),
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.#", "1"),
				),
			},
			// Step 4: Update by removing groups and adding users back (only the users field remains, groups field is removed)
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_updated_remove_group.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.0", "data.axual_user.ben", "id"),
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.#", "1"),
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.#", "0"),
				),
			},
			// Step 5: Update by adding both users and groups back
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_updated_add_both.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.0", "data.axual_user.ben", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.0", "axual_group.team-group3", "id"),
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.#", "1"),
					resource.TestCheckResourceAttr("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.#", "1"),
				),
			},
			// Step 6: Test validation - should error when both users and groups are missing
			{
				Config:      GetProvider() + GetFile("axual_topic_browse_permissions_updated_missing_groups_users.tf"),
				ExpectError: regexp.MustCompile(`Error message: either 'users' or 'groups' must be provided`),
			},
			// Step 7: Cleanup - ensure resources are destroyed properly
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_topic_browse_permissions_initial.tf"),
			},
		},
	})
}

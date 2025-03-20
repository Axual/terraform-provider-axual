package TopicBrowsePermissionsResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTopicBrowsePermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.0", "axual_user.ben", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.0", "axual_group.team-group", "id"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_topic_browse_permissions_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "users.0", "axual_user.chris", "id"),
					resource.TestCheckResourceAttrPair("axual_topic_browse_permissions.tf-test-topic-browse-permissions", "groups.0", "axual_group.team-group3", "id"),
				),
			},
			{
				Config:      GetProvider() + GetFile("axual_topic_browse_permissions_updated_missing_groups_users.tf"),
				ExpectError: regexp.MustCompile(`Error message: either 'users' or 'groups' must be provided`),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_topic_browse_permissions_initial.tf"),
			},
		},
	})
}

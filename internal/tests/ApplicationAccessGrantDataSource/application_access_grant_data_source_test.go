package ApplicationAccessGrantDataSource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationAccessGrantDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_access_grant.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.axual_application_access_grant.tf-test-application-access-grant-imported", "access_type", "CONSUMER"),
					resource.TestCheckResourceAttr("data.axual_application_access_grant.tf-test-application-access-grant-imported", "status", "Approved"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant-imported", "id", "axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant-imported", "environment", "axual_environment.tf-test-env", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant-imported", "topic", "axual_topic.tf-test-topic", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant-imported", "application", "axual_application.tf-test-app", "id"),
				),
			},
		},
	})
}

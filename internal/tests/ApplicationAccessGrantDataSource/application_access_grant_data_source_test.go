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
					resource.TestCheckResourceAttr("data.axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					resource.TestCheckResourceAttr("data.axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant", "id", "axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant", "environment", "axual_environment.tf-test-env", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant", "topic", "axual_topic.topic-test", "id"),
					resource.TestCheckResourceAttrPair("data.axual_application_access_grant.tf-test-application-access-grant", "application", "axual_application.tf-test-app", "id"),
				),
			},
		},
	})
}

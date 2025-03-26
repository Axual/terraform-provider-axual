package ApplicationResource

import (
	. "axual.com/terraform-provider-axual/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "name", "tf-test app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "short_name", "tf_test_app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_id", "tf.test.app"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "type", "Java"),
					resource.TestCheckResourceAttrPair("axual_application.tf-test-app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "visibility", "Public"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "description", "Axual's TF Test Application"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "name", "tf-test-app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_type", "Custom"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "short_name", "tf_test_app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "application_id", "tf.test.app1"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "type", "Pega"),
					resource.TestCheckResourceAttrPair("axual_application.tf-test-app", "owners", "data.axual_group.test_group", "id"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "visibility", "Private"),
					resource.TestCheckResourceAttr("axual_application.tf-test-app", "description", "Axual's TF Test Application1"),
				),
			},
			{
				ResourceName:      "axual_application.tf-test-app",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_updated.tf"),
			},
		},
	})
}

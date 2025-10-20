package ApplicationDeploymentResource

import (
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationDeploymentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_deployment_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_application_deployment.connector_axual_application_deployment", "environment", "axual_environment.tf-test-env", "id"),
					resource.TestCheckResourceAttrPair("axual_application_deployment.connector_axual_application_deployment", "application", "axual_application.tf-test-app", "id"),
					resource.TestCheckResourceAttr("axual_application_deployment.connector_axual_application_deployment", "configs.topic", "test-topic"),
					resource.TestCheckResourceAttr("axual_application_deployment.connector_axual_application_deployment", "configs.tasks.max", "1"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_deployment_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_application_deployment.connector_axual_application_deployment", "environment", "axual_environment.tf-test-env", "id"),
					resource.TestCheckResourceAttrPair("axual_application_deployment.connector_axual_application_deployment", "application", "axual_application.tf-test-app", "id"),
					resource.TestCheckResourceAttr("axual_application_deployment.connector_axual_application_deployment", "configs.topic", "test-topic"),
					resource.TestCheckResourceAttr("axual_application_deployment.connector_axual_application_deployment", "configs.tasks.max", "2"),
				),
			},
			{
				ResourceName:         "axual_application_deployment.connector_axual_application_deployment",
				ImportState:          true,
				ImportStateVerify:    true,
				Config:               GetProvider() + GetFile("axual_application_deployment_updated.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_deployment_updated.tf"),
			},
		},
	})
}

func TestApplicationDeploymentKSMLResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_deployment_ksml_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_application_deployment.ksml_axual_application_deployment", "environment", "axual_environment.tf-test-ksml-env", "id"),
					resource.TestCheckResourceAttrPair("axual_application_deployment.ksml_axual_application_deployment", "application", "axual_application.tf-test-ksml-app", "id"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "type", "KSML"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "deployment_size", "S"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "restart_policy", "on_exit"),
					resource.TestCheckResourceAttrSet("axual_application_deployment.ksml_axual_application_deployment", "definition"),
				),
			},
			{
				Config: GetProvider() + GetFile("axual_application_deployment_ksml_updated.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("axual_application_deployment.ksml_axual_application_deployment", "environment", "axual_environment.tf-test-ksml-env", "id"),
					resource.TestCheckResourceAttrPair("axual_application_deployment.ksml_axual_application_deployment", "application", "axual_application.tf-test-ksml-app", "id"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "type", "KSML"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "deployment_size", "M"),
					resource.TestCheckResourceAttr("axual_application_deployment.ksml_axual_application_deployment", "restart_policy", "on_exit"),
					resource.TestCheckResourceAttrSet("axual_application_deployment.ksml_axual_application_deployment", "definition"),
				),
			},
			{
				ResourceName:         "axual_application_deployment.ksml_axual_application_deployment",
				ImportState:          true,
				ImportStateVerify:    true,
				Config:               GetProvider() + GetFile("axual_application_deployment_ksml_updated.tf"),
			},
			{
				// To ensure cleanup if one of the test cases had an error
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_deployment_ksml_updated.tf"),
			},
		},
	})
}

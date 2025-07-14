package ApplicationDeploymentResource

import (
	"fmt"
	"testing"
	. "axual.com/terraform-provider-axual/internal/tests"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// extracts and validates application and environment IDs from state
func getResourceIDs(s *terraform.State) (applicationID, envID string, err error) {
	applicationResource, ok := s.RootModule().Resources["axual_application.tf-test-app"]
	if !ok {
		return "", "", fmt.Errorf("axual_application.tf-test-app resource not found in state")
	}

	envResource, ok := s.RootModule().Resources["axual_environment.tf-test-env"]
	if !ok {
		return "", "", fmt.Errorf("axual_environment.tf-test-env resource not found in state")
	}

	applicationID = applicationResource.Primary.ID
	envID = envResource.Primary.ID

	if applicationID == "" {
		return "", "", fmt.Errorf("application ID is empty")
	}

	if envID == "" {
		return "", "", fmt.Errorf("environment ID is empty")
	}

	return applicationID, envID, nil
}

// checkResourcesExist validates that required resources exist in the Terraform state
func checkResourcesExist(s *terraform.State) error {
	applicationID, envID, err := getResourceIDs(s)
	if err != nil {
		return err
	}

	// Log the IDs for debugging purposes
	fmt.Printf("ApplicationID: %s, EnvID: %s\n", applicationID, envID)
	return nil
}

// checkApplicationDeploymentExists validates that the application deployment resource exists
func checkApplicationDeploymentExists(s *terraform.State) error {
	_, ok := s.RootModule().Resources["axual_application_deployment.connector_axual_application_deployment"]
	if !ok {
		return fmt.Errorf("axual_application_deployment.connector_axual_application_deployment resource not found in state")
	}
	return nil
}

// importStateIdFunc generates the import ID from the current state
func importStateIdFunc(s *terraform.State) (string, error) {
	applicationID, envID, err := getResourceIDs(s)
	if err != nil {
		return "", err
	}

	importId := fmt.Sprintf("%s/%s", applicationID, envID)
	fmt.Printf("Import ID: %s\n", importId)
	return importId, nil
}

func TestAccApplicationDeploymentResourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile("axual_application_deployment_initial.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Check that all required resources exist in state
					checkResourcesExist,
					checkApplicationDeploymentExists,
					// Standard resource attribute checks
					resource.TestCheckResourceAttrSet("axual_application_deployment.connector_axual_application_deployment", "id"),
					resource.TestCheckResourceAttrSet("axual_application.tf-test-app", "id"),
					resource.TestCheckResourceAttrSet("axual_environment.tf-test-env", "id"),
				),
			},
			{
				ResourceName:         "axual_application_deployment.connector_axual_application_deployment",
				ImportState:          true,
				ImportStateIdFunc:    importStateIdFunc,
				ImportStateVerify:    true,
				Config:               GetProvider() + GetFile("axual_application_deployment_updated.tf"),
			},
		},
	})
}
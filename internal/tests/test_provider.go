package tests

import (
	"axual.com/terraform-provider-axual/internal/provider"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"testing"
)

// Struct to hold provider configuration from YAML file
type ProviderConfig struct {
	Provider struct {
		Version string `yaml:"version"` // Can be "local" or a version from the registry (e.g., "2.4.1")
	} `yaml:"provider"`
	InstanceName string `yaml:"instanceName"`
	UserGroup    string `yaml:"userGroup"`
}

// Function to load the configuration from a YAML file
func LoadProviderConfig() (ProviderConfig, error) {
	file, err := os.ReadFile("../test_config.yaml")
	if err != nil {
		return ProviderConfig{}, err
	}

	var config ProviderConfig
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return ProviderConfig{}, err
	}
	return config, nil
}

// This function helps select the appropriate provider configuration for tests.
func testProviderConfig() (resource.TestCase, error) {
	// Load the provider configuration from the YAML file
	config, err := LoadProviderConfig()
	if err != nil {
		return resource.TestCase{}, err
	}

	if config.Provider.Version == "local" {
		// Use local provider factories
		return resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories(),
		}, nil
	} else {
		// Use a specific version from the registry
		return resource.TestCase{
			ExternalProviders: map[string]resource.ExternalProvider{
				"axual": {
					VersionConstraint: config.Provider.Version,
					Source:            "Axual/axual",
				},
			},
		}, nil
	}
}

func GetProviderConfig(t *testing.T) resource.TestCase {
	providerConfig, err := testProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	return providerConfig
}

// Factory function for creating local provider instances
func testAccProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"axual": providerserver.NewProtocol6WithError(provider.New("dev")()),
	}
}

// Reusable provider configuration for resource creation
func GetProvider() string {
	// Load the provider configuration from the YAML file
	config, err := LoadProviderConfig()
	if err != nil {
		panic("Error loading provider config: " + err.Error())
	}
	// Local Platform.local setup
	providerBlock := `
	provider "axual" {
	 authmode = "keycloak"
	 apiurl   = "https://platform.local/api"
	 realm    = "local"
	 username = "` + os.Getenv("AXUAL_USERNAME") + `"
	 password = "` + os.Getenv("AXUAL_PASSWORD") + `"
	 clientid = "self-service"
	 authurl  = "https://platform.local/auth/realms/local/protocol/openid-connect/token"
	 scopes   = ["openid", "profile", "email"]
	}
	`

	// QA
	//	providerBlock := `
	//provider "axual" {
	//  authmode = "keycloak"
	//  apiurl   = "https://self-service.qa.np.westeurope.azure.axual.cloud/api"
	//  realm    = "axual"
	//  username = "` + os.Getenv("AXUAL_USERNAME") + `"
	//  password = "` + os.Getenv("AXUAL_PASSWORD") + `"
	//  clientid = "self-service"
	//  authurl  = "https://self-service.qa.np.westeurope.azure.axual.cloud/auth/realms/axual/protocol/openid-connect/token"
	//  scopes   = ["openid", "profile", "email"]
	//}
	//`

	dataBlock := `
	data "axual_instance" "testInstance" {
	  name = "` + config.InstanceName + `"
	}
	data "axual_group" "user_group" {
	  name = "` + config.UserGroup + `"
	}
	`

	// If the version is not "local", include the required_providers block with the version from the configuration file
	if config.Provider.Version != "local" {
		return `
terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = "` + config.Provider.Version + `"
    }
  }
}

` + providerBlock + dataBlock
	}

	// Return only the provider block if using the local provider
	return providerBlock + dataBlock
}

func GetFile(paths ...string) string {
	var combinedConfig string

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Panicf("Error reading %s: %s", path, err)
		}
		combinedConfig += string(data) + "\n" // Add file content and newline for separation
	}

	return combinedConfig
}

// Helper function to read the file and compare its content
func CheckBodyMatchesFile(resourceName, attrName, filePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Read the file
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		// Get the resource from the state
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("no resource in state")
		}

		// Get the value of the `body` attribute from the resource
		attrValue := rs.Primary.Attributes[attrName]

		// Compare the file content with the `body` attribute
		if string(fileContent) != attrValue {
			return fmt.Errorf("expected body to be '%s', got '%s'", string(fileContent), attrValue)
		}

		return nil
	}
}

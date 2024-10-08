package tests

import (
	"axual.com/terraform-provider-axual/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gopkg.in/yaml.v3"
	"os"
)

// Struct to hold provider configuration from YAML file
type ProviderConfig struct {
	Provider struct {
		Version string `yaml:"version"` // Can be "local" or a version from the registry (e.g., "2.4.1")
	} `yaml:"provider"`
}

// Function to load the configuration from a YAML file
func loadProviderConfig() (ProviderConfig, error) {
	file, err := os.ReadFile("test_config.yaml")
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
	config, err := loadProviderConfig()
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

// Factory function for creating local provider instances
func testAccProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"axual": providerserver.NewProtocol6WithError(provider.New("dev")()),
	}
}

// Reusable provider configuration for resource creation
func testAccProviderConfig() string {
	// Load the provider configuration from the YAML file
	config, err := loadProviderConfig()
	if err != nil {
		panic("Error loading provider config: " + err.Error())
	}

	providerBlock := `
provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "local"
  username = "` + os.Getenv("AXUAL_USERNAME") + `"
  password = "` + os.Getenv("AXUAL_PASSWORD") + `"
  clientid = "self-service"
  authurl  = "https://platform.local/auth/realms/local/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
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

` + providerBlock
	}

	// Return only the provider block if using the local provider
	return providerBlock
}

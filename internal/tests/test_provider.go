package tests

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	webclient "axual-webclient"

	"axual.com/terraform-provider-axual/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gopkg.in/yaml.v3"
)

// ProviderConfig Struct to hold provider configuration from the YAML file
type ProviderConfig struct {
	Provider struct {
		Version string `yaml:"version"` // Can be "local" or a version from the registry (e.g., "2.4.1")
	} `yaml:"provider"`
	ApiUrl            string `yaml:"apiUrl"`
	AuthUrl           string `yaml:"authUrl"`
	Realm             string `yaml:"realm"`
	InstanceName      string `yaml:"instanceName"`
	InstanceShortName string `yaml:"instanceShortName"`
	GroupName         string `yaml:"groupName"`
	UserEmail         string `yaml:"userEmail"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
}

// LoadProviderConfig Function to load the configuration from a YAML file
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
	if config.Realm == "" {
		config.Realm = "axual"
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

	// `time` is used by test fixtures to insert `time_sleep` between resources where the platform
	// has propagation lag (e.g. after activating an Application Principal, the search endpoint takes
	// a short moment to reflect the new active flag). Keeping the wait in test fixtures avoids
	// adding sleeps to the provider itself.
	timeProvider := map[string]resource.ExternalProvider{
		"time": {Source: "hashicorp/time"},
	}

	if config.Provider.Version == "local" {
		// Use local provider factories
		return resource.TestCase{
			ProtoV6ProviderFactories: testAccProviderFactories(),
			ExternalProviders:        timeProvider,
		}, nil
	}
	// Use a specific version from the registry
	external := map[string]resource.ExternalProvider{
		"axual": {
			VersionConstraint: config.Provider.Version,
			Source:            "Axual/axual",
		},
		"time": {Source: "hashicorp/time"},
	}
	return resource.TestCase{
		ExternalProviders: external,
	}, nil

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

// GetProvider Reusable provider configuration for resource creation
func GetProvider() string {
	// Load the provider configuration from the YAML file
	config, err := LoadProviderConfig()
	if err != nil {
		panic("Error loading provider config: " + err.Error())
	}

	providerBlock := `
		provider "axual" {
		authmode = "keycloak"
		apiurl   = "` + config.ApiUrl + `"
		realm    = "` + config.Realm + `"
		username = "` + config.Username + `"
		password = "` + config.Password + `"
		clientid = "self-service"
		authurl  = "` + config.AuthUrl + `"
		scopes   = ["openid", "profile", "email"]
	}
	`

	dataBlock := `
	data "axual_instance" "test_instance" {
	  name = "` + config.InstanceName + `"
	}
	data "axual_instance" "test_instance_by_short_name" {
	  short_name = "` + config.InstanceShortName + `"
	}
	data "axual_instance" "test_instance_by_name_and_short_name" {
	  name = "` + config.InstanceName + `"
	  short_name = "` + config.InstanceShortName + `"
	}
	data "axual_group" "test_group" {
	  name = "` + config.GroupName + `"
	}
	data "axual_user" "test_user" {
	  email = "` + config.UserEmail + `"
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

	return strings.ReplaceAll(combinedConfig, "{{CERTS}}", CertsDir())
}

// CertsDir returns the absolute path to the shared certs directory.
// Resolved relative to this source file so it works regardless of the caller's working directory.
func CertsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Panic("unable to determine CertsDir via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(file), "sharedCerts")
}

// CertPath returns the absolute path to a named cert in the shared certs directory.
func CertPath(name string) string {
	return filepath.Join(CertsDir(), name)
}

// apiClient builds an authenticated webclient against the configured platform, mirroring the
// provider block used by GetProvider (apiUrl/authUrl come from test_config.yaml). Used by check
// helpers that must inspect live API state that the provider deliberately does not refresh into
// Terraform state (e.g. a principal's activation status).
func apiClient() (*webclient.Client, error) {
	config, err := LoadProviderConfig()
	if err != nil {
		return nil, err
	}
	return webclient.NewClient(
		config.ApiUrl,
		config.Realm,
		webclient.AuthStruct{
			Username: config.Username,
			Password: config.Password,
			Url:      config.AuthUrl,
			ClientId: "self-service",
			Scopes:   []string{"openid", "profile", "email"},
			AuthMode: "keycloak",
		},
	)
}

// CheckPrincipalActiveInAPI asserts the LIVE API activation status of the principal backing the
// given resource (looked up by its state `id`). The provider treats `active` as write-only intent
// and never refreshes it from the API, so state-based checks cannot observe activation inherited
// across a rotation — this reads the API directly. Retries briefly to absorb activation
// propagation lag.
func CheckPrincipalActiveInAPI(resourceName string, wantActive bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("no resource %q in state", resourceName)
		}
		id := rs.Primary.Attributes["id"]
		if id == "" {
			return fmt.Errorf("resource %q has empty id in state", resourceName)
		}
		client, err := apiClient()
		if err != nil {
			return fmt.Errorf("unable to build API client: %w", err)
		}
		var lastActive bool
		for attempt := 0; attempt < 5; attempt++ {
			p, err := client.ReadApplicationPrincipal(id)
			if err != nil {
				return fmt.Errorf("unable to read principal %s from API: %w", id, err)
			}
			lastActive = p.Active != nil && *p.Active
			if lastActive == wantActive {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
		return fmt.Errorf("principal %q (%s): expected API active=%t, got %t", resourceName, id, wantActive, lastActive)
	}
}

// CheckBodyMatchesFile Helper function to read the file and compare its content
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

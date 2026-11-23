package tests

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
	// A service account uid to use as a group member.
	ServiceAccountUid string `yaml:"serviceAccountUid"`
	// The email of a user who logged in once and may have its roles changed: TestUserResource imports it.
	TestUserEmail string `yaml:"testUserEmail"`
	// IamGroupTests turns on the IAM group tests. Missing means false.
	IamGroupTests bool `yaml:"iamGroupTests"`
	// Strict makes a test fail instead of skip when a key it needs is empty, so a run with an incomplete
	// file cannot pass. Tests for a feature of a newer Platform Manager still skip. Missing means false.
	Strict bool `yaml:"strict"`
	// FlinkTests = false skips the Flink tests on purpose, also with strict: true (for a stack without
	// Ververica). Missing means the Flink tests run when the Ververica values are set.
	FlinkTests *bool `yaml:"flinkTests"`
	// The Ververica values the Flink suites need. The API validates the workspace, namespace and
	// deployment target live, so a developer whose namespace differs sets them here rather than
	// editing the fixtures; they default to defaultworkspace / default / default-target.
	VervericaUrl              string `yaml:"ververicaUrl"`
	VervericaApiToken         string `yaml:"ververicaApiToken"`
	ClusterId                 string `yaml:"clusterId"`
	VervericaWorkspace        string `yaml:"ververicaWorkspace"`
	VervericaNamespace        string `yaml:"ververicaNamespace"`
	VervericaDeploymentTarget string `yaml:"ververicaDeploymentTarget"`
	// ResolvedTopicPrefix is only needed by the hand-written Flink table DDL, whose
	// `CREATE TEMPORARY TABLE` names the real Kafka topic. That test skips when it is empty.
	ResolvedTopicPrefix string `yaml:"resolvedTopicPrefix"`
	// Kafka Connect cluster test values. A Kafka Connect cluster is registered by an administrator,
	// not created by Terraform - see the axual_connect_cluster data source - so these name two
	// clusters that must already exist: one MTLS-authenticated, one SASL_SCRAM-authenticated, both
	// on the same instance/cluster.
	ConnectClusterInstanceId string `yaml:"connectClusterInstanceId"`
	ConnectClusterClusterId  string `yaml:"connectClusterClusterId"`
	ConnectClusterMtlsId     string `yaml:"connectClusterMtlsId"`
	ConnectClusterMtlsName   string `yaml:"connectClusterMtlsName"`
	ConnectClusterSaslId     string `yaml:"connectClusterSaslId"`
	ConnectClusterSaslName   string `yaml:"connectClusterSaslName"`
	// The Kafka cluster and schema registry the Cluster and Instance tests use (see DEVELOPERS.md).
	KafkaBootstrapServers      string `yaml:"kafkaBootstrapServers"`
	KafkaClientCertificateFile string `yaml:"kafkaClientCertificateFile"`
	KafkaClientPrivateKeyFile  string `yaml:"kafkaClientPrivateKeyFile"`
	KafkaTruststoreFile        string `yaml:"kafkaTruststoreFile"`
	// A PEM file of the CA that signs application certificates, for the Instance's ca_certificates.
	InstanceCaCertificateFile string `yaml:"instanceCaCertificateFile"`
	// An Apicurio registry for the Instance-Cluster tests.
	SchemaRegistryUrl      string `yaml:"schemaRegistryUrl"`
	SchemaRegistryUsername string `yaml:"schemaRegistryUsername"`
	SchemaRegistryPassword string `yaml:"schemaRegistryPassword"`
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
	if config.VervericaWorkspace == "" {
		config.VervericaWorkspace = "defaultworkspace"
	}
	if config.VervericaNamespace == "" {
		config.VervericaNamespace = "default"
	}
	if config.VervericaDeploymentTarget == "" {
		config.VervericaDeploymentTarget = "default-target"
	}
	return config, nil
}

// SkipForMissingConfig skips a test because a test_config.yaml value it needs is not set. With
// strict: true the test fails instead: a package whose tests all skip still reports ok, which can hide
// an incomplete file.
func SkipForMissingConfig(t *testing.T, config ProviderConfig, format string, args ...any) {
	t.Helper()
	if config.Strict {
		t.Fatalf("strict: true in test_config.yaml, and "+format+". Set it, or remove strict: true", args...)
	}
	t.Skipf(format, args...)
}

// missing reports whether a test_config.yaml value is empty or still a YOUR_ placeholder.
func missing(value string) bool {
	return value == "" || strings.HasPrefix(value, "YOUR_")
}

// SkipWithoutVerverica skips a Flink test unless test_config.yaml carries real Ververica values.
// The committed file ships `YOUR_...` placeholders, so without this the Flink suites fail rather
// than skip for every developer who has no Ververica namespace. flinkTests: false skips them on purpose.
func SkipWithoutVerverica(t *testing.T, config ProviderConfig) {
	t.Helper()
	if config.FlinkTests != nil && !*config.FlinkTests {
		t.Skip("flinkTests is false in test_config.yaml")
	}
	for _, field := range []struct{ key, value string }{
		{"ververicaUrl", config.VervericaUrl},
		{"ververicaApiToken", config.VervericaApiToken},
		{"clusterId", config.ClusterId},
	} {
		if missing(field.value) {
			SkipForMissingConfig(t, config, "%s is not set in test_config.yaml, so no Flink Cluster can be registered", field.key)
		}
	}
}

// SkipWithoutConnectCluster skips a Kafka Connect test unless test_config.yaml names two already
// registered Kafka Connect clusters (one MTLS, one SASL_SCRAM). Unlike a Flink Cluster, these are
// never created by this suite - registering one is an administrator action outside Terraform - so
// the committed placeholders always skip rather than fail for a developer whose stack has none.
func SkipWithoutConnectCluster(t *testing.T, config ProviderConfig) {
	t.Helper()
	for _, field := range []struct{ key, value string }{
		{"connectClusterInstanceId", config.ConnectClusterInstanceId},
		{"connectClusterClusterId", config.ConnectClusterClusterId},
		{"connectClusterMtlsId", config.ConnectClusterMtlsId},
		{"connectClusterMtlsName", config.ConnectClusterMtlsName},
		{"connectClusterSaslId", config.ConnectClusterSaslId},
		{"connectClusterSaslName", config.ConnectClusterSaslName},
	} {
		if missing(field.value) {
			skipBefore(t, config, 16, 0, "registered Kafka Connect clusters")
			SkipForMissingConfig(t, config, "%s is not set in test_config.yaml, so no registered Kafka Connect cluster is available", field.key)
		}
	}
}

// SkipWithoutInfiniteRetention skips a test that sets retention_time = -1 on a Platform Manager that
// refuses it. It signs in, so call it from PreCheck, which only runs with TF_ACC set.
func SkipWithoutInfiniteRetention(t *testing.T) {
	t.Helper()
	version, err := platformVersion()
	if err != nil {
		t.Fatalf("Error reading the Platform Manager version: %v", err)
	}
	if !provider.SupportsInfiniteRetention(version) {
		t.Skipf("Platform Manager %s does not accept retention_time = -1", version)
	}
}

// SkipStepWithoutInstanceFlink is the SkipFunc of a test step that sets flink_enabled = true on an
// axual_instance: it skips the step on a Platform Manager before 16.0.0, which does not have it.
func SkipStepWithoutInstanceFlink() (bool, error) {
	return skipStepUnless(provider.SupportsInstanceFlink, "flink_enabled on an Instance")
}

// SkipStepWithoutSchemaRegistryApi is the SkipFunc of a test step that sets api or client_endpoints on
// an axual_instance_cluster: it skips the step on a Platform Manager before 16.1.0, which does not have
// them.
func SkipStepWithoutSchemaRegistryApi() (bool, error) {
	return skipStepUnless(provider.SupportsSchemaRegistryApi, "api and client endpoints of a schema registry")
}

// SkipStepBefore returns the SkipFunc of a test step that uses a feature of Platform Manager major.minor
// or later: it skips the step on an older Platform Manager.
func SkipStepBefore(major, minor int, feature string) func() (bool, error) {
	return func() (bool, error) {
		return skipStepUnless(func(version string) bool { return versionAtLeast(version, major, minor) },
			fmt.Sprintf("%s (needs %d.%d.0)", feature, major, minor))
	}
}

// CheckFrom runs check only on Platform Manager major.minor or later, for a value an older Platform
// Manager does not have.
func CheckFrom(major, minor int, check resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		version, err := platformVersion()
		if err != nil {
			return err
		}
		if !versionAtLeast(version, major, minor) {
			log.Printf("[WARN] Platform Manager %s is older than %d.%d.0: this check is skipped", version, major, minor)
			return nil
		}
		return check(s)
	}
}

// SkipWithoutServiceAccount skips a test that adds a service account to a group unless
// test_config.yaml names one. Platform Manager before 16.0.0 has no service accounts.
func SkipWithoutServiceAccount(t *testing.T, config ProviderConfig) {
	t.Helper()
	if missing(config.ServiceAccountUid) {
		skipBefore(t, config, 16, 0, "service accounts")
		SkipForMissingConfig(t, config, "serviceAccountUid is not set in test_config.yaml, so no service account can be added to a group")
	}
}

// skipBefore skips the test, with strict: true only, when the Platform Manager is older than
// major.minor and so has no feature: then an empty test_config.yaml value is expected. Without strict
// it signs in to nothing, because the test skips for the empty value anyway.
func skipBefore(t *testing.T, config ProviderConfig, major, minor int, feature string) {
	t.Helper()
	if !config.Strict {
		return
	}
	version, err := platformVersion()
	if err != nil {
		t.Fatalf("Error reading the Platform Manager version: %v", err)
	}
	if !versionAtLeast(version, major, minor) {
		t.Skipf("Platform Manager %s has no %s (needs %d.%d.0)", version, feature, major, minor)
	}
}

// platformVersion signs in and reads the Platform Manager version.
func platformVersion() (string, error) {
	client, err := apiClient()
	if err != nil {
		return "", fmt.Errorf("signing in to read the Platform Manager version: %w", err)
	}
	version, err := client.GetPlatformVersion()
	if err != nil {
		return "", fmt.Errorf("reading the Platform Manager version: %w", err)
	}
	return version, nil
}

// versionAtLeast reports whether a Platform Manager version such as "16.0.4" or "16.1.0-SNAPSHOT" is
// major.minor or later. A version it cannot read counts as too old, as in the provider.
func versionAtLeast(version string, major, minor int) bool {
	var gotMajor, gotMinor int
	if _, err := fmt.Sscanf(version, "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false
	}
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

// skipStepUnless skips a test step when the Platform Manager version does not support a feature. Test
// steps only run with TF_ACC set, so it may sign in.
func skipStepUnless(supports func(version string) bool, feature string) (bool, error) {
	version, err := platformVersion()
	if err != nil {
		return false, err
	}
	if !supports(version) {
		log.Printf("[WARN] Platform Manager %s has no %s: this step is skipped", version, feature)
		return true, nil
	}
	return false, nil
}

// SkipWithoutKafkaCluster skips a Cluster or Instance test unless test_config.yaml names a Kafka
// cluster Terraform can register.
func SkipWithoutKafkaCluster(t *testing.T, config ProviderConfig) {
	t.Helper()
	for _, field := range []struct{ key, value string }{
		{"kafkaBootstrapServers", config.KafkaBootstrapServers},
		{"kafkaClientCertificateFile", config.KafkaClientCertificateFile},
		{"kafkaClientPrivateKeyFile", config.KafkaClientPrivateKeyFile},
		{"kafkaTruststoreFile", config.KafkaTruststoreFile},
		{"instanceCaCertificateFile", config.InstanceCaCertificateFile},
	} {
		if missing(field.value) {
			SkipForMissingConfig(t, config, "%s is not set in test_config.yaml, so no Kafka cluster can be registered", field.key)
		}
	}
}

// SkipWithoutSchemaRegistry skips an Instance-Cluster test unless test_config.yaml names an
// Apicurio registry.
func SkipWithoutSchemaRegistry(t *testing.T, config ProviderConfig) {
	t.Helper()
	SkipWithoutKafkaCluster(t, config)
	for _, field := range []struct{ key, value string }{
		{"schemaRegistryUrl", config.SchemaRegistryUrl},
		{"schemaRegistryUsername", config.SchemaRegistryUsername},
		{"schemaRegistryPassword", config.SchemaRegistryPassword},
	} {
		if missing(field.value) {
			SkipForMissingConfig(t, config, "%s is not set in test_config.yaml, so no schema registry is available", field.key)
		}
	}
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

// GetFlinkProvider returns GetProvider() plus the locals the Flink fixtures read. Only the Flink
// suites call it, so the Ververica API token is not written into the generated HCL of every other
// suite, where it would show up in saved plans and TF_LOG output.
func GetFlinkProvider() string {
	config, err := LoadProviderConfig()
	if err != nil {
		log.Panicf("Error loading provider configuration: %s", err)
	}

	return GetProvider() + `
	locals {
	  cluster_id = "` + config.ClusterId + `"
	  ververica_url = "` + config.VervericaUrl + `"
	  ververica_api_token = "` + config.VervericaApiToken + `"
	  resolved_topic_prefix = "` + config.ResolvedTopicPrefix + `"
	  ververica_workspace = "` + config.VervericaWorkspace + `"
	  ververica_namespace = "` + config.VervericaNamespace + `"
	  ververica_deployment_target = "` + config.VervericaDeploymentTarget + `"
	}
	`
}

// GetConnectProvider returns GetProvider() plus the locals the Kafka Connect fixtures read: the
// instance/cluster a registered Kafka Connect cluster lives on, and the ids/names of the two test
// clusters (MTLS and SASL_SCRAM) an administrator must have already registered there.
func GetConnectProvider() string {
	config, err := LoadProviderConfig()
	if err != nil {
		log.Panicf("Error loading provider configuration: %s", err)
	}

	return GetProvider() + `
	locals {
	  connect_cluster_instance_id = "` + config.ConnectClusterInstanceId + `"
	  connect_cluster_cluster_id = "` + config.ConnectClusterClusterId + `"
	  connect_cluster_mtls_id = "` + config.ConnectClusterMtlsId + `"
	  connect_cluster_mtls_name = "` + config.ConnectClusterMtlsName + `"
	  connect_cluster_sasl_id = "` + config.ConnectClusterSaslId + `"
	  connect_cluster_sasl_name = "` + config.ConnectClusterSaslName + `"
	}
	`
}

// GetClusterProvider returns GetProvider() plus the locals the Cluster, Instance and
// Instance-Cluster fixtures read.
func GetClusterProvider() string {
	config, err := LoadProviderConfig()
	if err != nil {
		log.Panicf("Error loading provider configuration: %s", err)
	}

	return GetProvider() + `
	locals {
	  kafka_bootstrap_servers = "` + config.KafkaBootstrapServers + `"
	  kafka_client_certificate_file = "` + config.KafkaClientCertificateFile + `"
	  kafka_client_private_key_file = "` + config.KafkaClientPrivateKeyFile + `"
	  kafka_truststore_file = "` + config.KafkaTruststoreFile + `"
	  instance_ca_certificate_file = "` + config.InstanceCaCertificateFile + `"
	  schema_registry_url = "` + config.SchemaRegistryUrl + `"
	  schema_registry_username = "` + config.SchemaRegistryUsername + `"
	  schema_registry_password = "` + config.SchemaRegistryPassword + `"
	}
	`
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
// ApiClient is apiClient for the test packages, for a step that has to change platform state
// outside Terraform (for example stopping a deployment to test drift).
func ApiClient() (*webclient.Client, error) {
	return apiClient()
}

func apiClient() (*webclient.Client, error) {
	config, err := LoadProviderConfig()
	if err != nil {
		return nil, err
	}
	return webclient.NewClient(
		config.ApiUrl,
		config.Realm,
		webclient.Credentials{
			Mode:     webclient.ModeROPC,
			TokenURL: config.AuthUrl,
			ClientID: "self-service",
			Username: config.Username,
			Password: config.Password,
			Scopes:   []string{"openid", "profile", "email"},
		},
	)
}

// RequireIamGroupManagement runs an IAM group test only when test_config.yaml has iamGroupTests: true, and skips it
// otherwise, before any call to Platform Manager. With the switch on, it switches IAM group management on for the test
// and off again afterwards, and it fails, and does not skip, when it cannot. A stopped run (Ctrl-C or -timeout) leaves
// the setting on: switch it off in the tenant settings then.
func RequireIamGroupManagement(t *testing.T) {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	config, err := LoadProviderConfig()
	if err != nil {
		t.Fatalf("Error loading provider config: %v", err)
	}
	if !config.IamGroupTests {
		t.Skip("iamGroupTests is not true in test_config.yaml, so the IAM group tests are skipped")
	}
	client, err := apiClient()
	if err != nil {
		t.Fatalf("unable to build API client: %v", err)
	}
	// The tenant of the signed-in user: the same tenant that the settings call below changes.
	var tenant struct {
		Settings map[string]any `json:"settings"`
	}
	if err := client.RequestAndMap("GET", client.ApiURL+"/tenant", nil, map[string]string{"Accept": "application/hal+json"}, &tenant); err != nil {
		t.Fatalf("unable to read the tenant settings: %v", err)
	}
	settings := tenant.Settings
	if fmt.Sprint(settings["axualGroupManagementEnabled"]) == "false" {
		t.Fatal("iamGroupTests is true, but the tenant has Axual group management off, and the IAM group tests also need it: switch it on in the tenant settings")
	}
	if fmt.Sprint(settings["iamGroupManagementEnabled"]) == "true" {
		return
	}
	post := func(body string) error {
		return client.RequestAndMap("POST", client.ApiURL+"/tenant/settings/group-management", strings.NewReader(body), map[string]string{"Content-Type": "application/json"}, nil)
	}
	automatic := fmt.Sprint(settings["autoIAMGroupCreationEnabled"]) == "true"
	if err := post(`{"iamGroupManagement":{"enabled":true,"automaticGroupCreationEnabled":false}}`); err != nil {
		var httpErr *webclient.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden {
			t.Fatal("iamGroupTests is true, but IAM group management is off and only a Tenant Admin can switch it on: use a Tenant Admin in test_config.yaml, or switch it on in the tenant settings")
		}
		t.Fatalf("unable to enable IAM group management: %v", err)
	}
	t.Cleanup(func() {
		// The automatic setting can only be changed while IAM group management is on.
		if automatic {
			if err := post(`{"iamGroupManagement":{"enabled":true,"automaticGroupCreationEnabled":true}}`); err != nil {
				t.Errorf("unable to put automatic IAM group creation back: %v", err)
			}
		}
		if err := post(`{"iamGroupManagement":{"enabled":false}}`); err != nil {
			t.Errorf("unable to switch IAM group management off again: %v", err)
		}
	})
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

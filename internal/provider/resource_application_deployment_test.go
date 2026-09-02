package provider

import (
	webclient "axual-webclient"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestShouldStopDeployment asserts that the decision to stop a deployment follows the `stop`
// link advertised by GET /application_deployments/{uid}/status. The fixtures are real API
// responses for the three application types, in a not-running and a running state.
func TestShouldStopDeployment(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		expected bool
	}{
		{name: "connector not running", fixture: "deployment_status_connector_stopped.json", expected: false},
		{name: "connector running", fixture: "deployment_status_connector_running.json", expected: true},
		{name: "flink undeployed", fixture: "deployment_status_flink_stopped.json", expected: false},
		{name: "flink starting", fixture: "deployment_status_flink_running.json", expected: true},
		{name: "ksml undeployed", fixture: "deployment_status_ksml_stopped.json", expected: false},
		{name: "ksml starting", fixture: "deployment_status_ksml_running.json", expected: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := readStatusFixture(t, test.fixture)
			if got := shouldStopDeployment(status); got != test.expected {
				t.Errorf("shouldStopDeployment() = %t, expected %t", got, test.expected)
			}
		})
	}
}

// TestShouldStopDeploymentWithoutLinks covers responses that carry no usable `_links`: without a
// `stop` link there is no valid STOP action, so the deployment must not be stopped.
func TestShouldStopDeploymentWithoutLinks(t *testing.T) {
	for _, body := range []string{`{}`, `{"_links":{}}`} {
		var status webclient.ApplicationDeploymentStatusResponse
		if err := json.Unmarshal([]byte(body), &status); err != nil {
			t.Fatalf("unable to unmarshal %s: %s", body, err)
		}
		if shouldStopDeployment(&status) {
			t.Errorf("shouldStopDeployment() = true for %s, expected false", body)
		}
	}
}

func readStatusFixture(t *testing.T, name string) *webclient.ApplicationDeploymentStatusResponse {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("unable to read fixture %s: %s", name, err)
	}

	var status webclient.ApplicationDeploymentStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("unable to unmarshal fixture %s: %s", name, err)
	}
	return &status
}

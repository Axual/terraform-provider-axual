package provider

import (
	webclient "axual-webclient"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
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

// TestIsFlinkTaskSizeOnlyChange covers which updates take the API's `flink_task_size`-only patch
// path: only a FLINK_SQL deployment whose size is the single difference between state and plan.
func TestIsFlinkTaskSizeOnlyChange(t *testing.T) {
	flinkState := func() ApplicationDeploymentResourceData {
		return ApplicationDeploymentResourceData{
			Type:              types.StringValue("FLINK_SQL"),
			DeploymentSize:    types.StringValue("S"),
			SqlScript:         types.StringValue("INSERT INTO out SELECT * FROM in;"),
			GenerateTablesSql: types.BoolValue(true),
		}
	}

	tests := []struct {
		name     string
		plan     func(*ApplicationDeploymentResourceData)
		expected bool
	}{
		{
			name:     "size only",
			plan:     func(d *ApplicationDeploymentResourceData) { d.DeploymentSize = types.StringValue("M") },
			expected: true,
		},
		{
			name:     "nothing changed",
			plan:     func(d *ApplicationDeploymentResourceData) {},
			expected: false,
		},
		{
			name: "size and sql",
			plan: func(d *ApplicationDeploymentResourceData) {
				d.DeploymentSize = types.StringValue("M")
				d.SqlScript = types.StringValue("INSERT INTO out SELECT `key` FROM in;")
			},
			expected: false,
		},
		{
			name: "size and generate_tables_sql",
			plan: func(d *ApplicationDeploymentResourceData) {
				d.DeploymentSize = types.StringValue("M")
				d.GenerateTablesSql = types.BoolValue(false)
			},
			expected: false,
		},
		{
			name: "unknown size",
			plan: func(d *ApplicationDeploymentResourceData) {
				d.DeploymentSize = types.StringUnknown()
			},
			expected: false,
		},
		{
			name: "ksml size change",
			plan: func(d *ApplicationDeploymentResourceData) {
				d.Type = types.StringValue("Ksml")
				d.DeploymentSize = types.StringValue("M")
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := flinkState()
			plan := flinkState()
			if test.name == "ksml size change" {
				state.Type = types.StringValue("Ksml")
			}
			test.plan(&plan)

			if got := isFlinkTaskSizeOnlyChange(&state, &plan); got != test.expected {
				t.Errorf("isFlinkTaskSizeOnlyChange() = %t, expected %t", got, test.expected)
			}
		})
	}
}

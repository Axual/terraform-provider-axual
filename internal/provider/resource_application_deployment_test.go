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
		name           string
		deploymentType string
		fixture        string
		expected       bool
	}{
		{name: "connector not running", deploymentType: "Connector", fixture: "deployment_status_connector_stopped.json", expected: false},
		{name: "connector running", deploymentType: "Connector", fixture: "deployment_status_connector_running.json", expected: true},
		{name: "flink undeployed", deploymentType: webclient.FlinkSQLApplicationType, fixture: "deployment_status_flink_stopped.json", expected: false},
		{name: "flink starting", deploymentType: webclient.FlinkSQLApplicationType, fixture: "deployment_status_flink_starting.json", expected: true},
		{name: "ksml undeployed", deploymentType: "Ksml", fixture: "deployment_status_ksml_stopped.json", expected: false},
		{name: "ksml starting", deploymentType: "Ksml", fixture: "deployment_status_ksml_starting.json", expected: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := readStatusFixture(t, test.fixture)
			if got := shouldStopDeployment(test.deploymentType, status); got != test.expected {
				t.Errorf("shouldStopDeployment() = %t, expected %t", got, test.expected)
			}
		})
	}
}

// TestShouldStopDeploymentWithoutLinks covers responses that carry no usable `_links`. No rels does
// not mean "already stopped" - a missing APPLICATION_DEPLOYMENT_UPDATE permission, an unavailable
// provisioner and an unknown Ververica state all report none - so the reported status decides.
func TestShouldStopDeploymentWithoutLinks(t *testing.T) {
	tests := []struct {
		name           string
		deploymentType string
		body           string
		expected       bool
	}{
		{name: "no links at all", deploymentType: "Connector", body: `{}`, expected: false},
		{name: "empty links", deploymentType: "Connector", body: `{"_links":{}}`, expected: false},
		{name: "connector running", deploymentType: "Connector", body: `{"connectorState":{"state":"Running"}}`, expected: true},
		{name: "ksml running", deploymentType: "Ksml", body: `{"ksmlStatus":{"status":"Running"}}`, expected: true},
		{name: "ksml undeployed", deploymentType: "Ksml", body: `{"ksmlStatus":{"status":"Undeployed"}}`, expected: false},
		{name: "flink running", deploymentType: webclient.FlinkSQLApplicationType, body: `{"flinkStatus":{"status":"Running"}}`, expected: true},
		{name: "flink undeployed", deploymentType: webclient.FlinkSQLApplicationType, body: `{"flinkStatus":{"status":"Undeployed"}}`, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var status webclient.ApplicationDeploymentStatusResponse
			if err := json.Unmarshal([]byte(test.body), &status); err != nil {
				t.Fatalf("unable to unmarshal %s: %s", test.body, err)
			}
			if got := shouldStopDeployment(test.deploymentType, &status); got != test.expected {
				t.Errorf("shouldStopDeployment() = %t for %s, expected %t", got, test.body, test.expected)
			}
		})
	}
}

// TestIsDeploymentStopped pins the states each application type is allowed to be deleted from. The
// values are the ones the API reports: KSML never says `Stopped` (a stopped app is `Undeployed`),
// and a Connector that never ran says `Undefined`.
func TestIsDeploymentStopped(t *testing.T) {
	tests := []struct {
		name           string
		deploymentType string
		status         string
		expected       bool
	}{
		{name: "connector stopped", deploymentType: "Connector", status: "Stopped", expected: true},
		{name: "connector never ran", deploymentType: "Connector", status: "Undefined", expected: true},
		{name: "connector failed", deploymentType: "Connector", status: "Failed", expected: true},
		{name: "connector running", deploymentType: "Connector", status: "Running", expected: false},
		{name: "connector starting", deploymentType: "Connector", status: "Starting", expected: false},
		{name: "ksml undeployed", deploymentType: "Ksml", status: "Undeployed", expected: true},
		{name: "ksml failed", deploymentType: "Ksml", status: "Failed", expected: true},
		{name: "ksml completed", deploymentType: "Ksml", status: "Completed", expected: true},
		{name: "ksml running", deploymentType: "Ksml", status: "Running", expected: false},
		{name: "ksml starting", deploymentType: "Ksml", status: "Starting", expected: false},
		{name: "flink undeployed", deploymentType: webclient.FlinkSQLApplicationType, status: "Undeployed", expected: true},
		{name: "flink failed", deploymentType: webclient.FlinkSQLApplicationType, status: "Failed", expected: true},
		{name: "flink running", deploymentType: webclient.FlinkSQLApplicationType, status: "Running", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := statusOf(test.deploymentType, test.status)
			if got := isDeploymentStopped(test.deploymentType, status); got != test.expected {
				t.Errorf("isDeploymentStopped(%s, %s) = %t, expected %t", test.deploymentType, test.status, got, test.expected)
			}
		})
	}
}

// TestIsDeploymentRunning covers the other half of the pair: only `Running` counts as running, for
// every application type.
func TestIsDeploymentRunning(t *testing.T) {
	for _, deploymentType := range []string{"Connector", "Ksml", webclient.FlinkSQLApplicationType} {
		for _, test := range []struct {
			status   string
			expected bool
		}{
			{status: "Running", expected: true},
			{status: "Starting", expected: false},
			{status: "Failed", expected: false},
			{status: "", expected: false},
		} {
			t.Run(deploymentType+" "+test.status, func(t *testing.T) {
				status := statusOf(deploymentType, test.status)
				if got := isDeploymentRunning(deploymentType, status); got != test.expected {
					t.Errorf("isDeploymentRunning(%s, %q) = %t, expected %t", deploymentType, test.status, got, test.expected)
				}
			})
		}
	}
}

// statusOf builds a status response that reports `status` on the field belonging to the type.
func statusOf(deploymentType string, status string) *webclient.ApplicationDeploymentStatusResponse {
	response := &webclient.ApplicationDeploymentStatusResponse{}
	switch {
	case isFlinkSQL(deploymentType):
		response.FlinkStatus.Status = status
	case isKSML(deploymentType):
		response.KsmlStatus.Status = status
	default:
		response.ConnectorState.State = status
	}
	return response
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
		state    func(*ApplicationDeploymentResourceData)
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
			name:  "ksml size change",
			state: func(d *ApplicationDeploymentResourceData) { d.Type = types.StringValue("Ksml") },
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
			if test.state != nil {
				test.state(&state)
			}
			plan := flinkState()
			test.plan(&plan)

			if got := isFlinkTaskSizeOnlyChange(&state, &plan); got != test.expected {
				t.Errorf("isFlinkTaskSizeOnlyChange() = %t, expected %t", got, test.expected)
			}
		})
	}
}

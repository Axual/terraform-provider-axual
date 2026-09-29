package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeConnector is a Platform Manager that knows one Connector deployment. Its live state follows
// ConnectorStateMachine.actionsFor: the `_links` it advertises depend on the state, and START,
// STOP, RESTART and RESET move the state the way Kafka Connect does.
type fakeConnector struct {
	mu         sync.Mutex
	state      string // Undefined, Starting, Running, Stopped, Failed
	taskStatus string // "" for no tasks, else Running or Failed
	trace      string
	// startsIn is how many status reads a START stays in Starting before it runs.
	startsIn  int
	failAfter bool // a START ends in Failed instead of Running
	gone      bool // the deployment does not exist
	actions   []string
}

func (f *fakeConnector) links() map[string]webclient.Link {
	rels := map[string][]string{
		"Undefined": {"start"},
		"Starting":  {"stop"},
		"Running":   {"stop", "restart"},
		"Failed":    {"stop", "restart"},
		"Stopped":   {"start", "reset"},
	}[f.state]
	links := map[string]webclient.Link{}
	for _, rel := range rels {
		links[rel] = webclient.Link{Href: "http://pm/" + rel}
	}
	return links
}

func (f *fakeConnector) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gone {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
		if f.state == "Starting" {
			if f.startsIn <= 0 {
				f.state = "Running"
				if f.failAfter {
					f.state = "Failed"
				}
			}
			f.startsIn--
		}
		body := map[string]any{
			"connectorState": map[string]any{"state": f.state, "trace": f.trace},
			"_links":         f.links(),
		}
		if f.taskStatus != "" {
			body["taskStates"] = []map[string]any{{"id": 0, "status": f.taskStatus, "trace": f.trace}}
		}
		_ = json.NewEncoder(w).Encode(body)
	case r.Method == http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":       "dep1",
			"_embedded": map[string]any{"application": map[string]any{"applicationType": "Connector"}},
		})
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/operation"):
		action := r.URL.Query().Get("action")
		f.actions = append(f.actions, action)
		switch action {
		case "START":
			f.state = "Starting"
		case "RESTART":
			f.state, f.taskStatus, f.trace = "Running", "Running", ""
		case "STOP":
			f.state = "Stopped"
		case "RESET":
			f.state = "Undefined"
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (f *fakeConnector) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.actions...)
}

// newStateResourceAgainst returns a state resource talking to fake, with every wait shortened.
func newStateResourceAgainst(t *testing.T, fake *fakeConnector) *applicationDeploymentStateResource {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(server.Close)

	for _, restore := range []func(){
		shorten(&runningWaitDelay), shorten(&resetWaitDelay), shorten(&stopWaitDelay), shorten(&startWaitDelay),
	} {
		t.Cleanup(restore)
	}
	client := &webclient.Client{HTTPClient: server.Client(), ApiURL: server.URL}
	return &applicationDeploymentStateResource{provider: AxualProvider{client: client}}
}

func shorten(d *time.Duration) func() {
	old := *d
	*d = time.Millisecond
	return func() { *d = old }
}

func equalActions(got []string, want ...string) bool {
	return strings.Join(got, ",") == strings.Join(want, ",")
}

func TestStateResourceRunStartsAnUndeployedConnector(t *testing.T) {
	fake := &fakeConnector{state: "Undefined", startsIn: 2}
	r := newStateResourceAgainst(t, fake)

	if err := r.run(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !equalActions(fake.sent(), "START") {
		t.Errorf("actions = %v, expected START", fake.sent())
	}
}

func TestStateResourceRunRestartsAFailedConnector(t *testing.T) {
	fake := &fakeConnector{state: "Failed", trace: "boom"}
	r := newStateResourceAgainst(t, fake)

	if err := r.run(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !equalActions(fake.sent(), "RESTART") {
		t.Errorf("actions = %v, expected RESTART", fake.sent())
	}
}

func TestStateResourceRunRestartsAConnectorWithAFailedTask(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Failed", trace: "task died"}
	r := newStateResourceAgainst(t, fake)

	if err := r.run(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !equalActions(fake.sent(), "RESTART") {
		t.Errorf("actions = %v, expected RESTART", fake.sent())
	}
}

func TestStateResourceRunLeavesARunningConnectorAlone(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Running"}
	r := newStateResourceAgainst(t, fake)

	if err := r.run(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(fake.sent()) != 0 {
		t.Errorf("actions = %v, expected none", fake.sent())
	}
}

func TestStateResourceRunReportsAConnectorThatFailsAfterStart(t *testing.T) {
	fake := &fakeConnector{state: "Stopped", failAfter: true, trace: "org.apache.kafka.common.errors.TopicAuthorizationException\n\tat ..."}
	r := newStateResourceAgainst(t, fake)

	err := r.run(context.Background(), "dep1", "Connector")
	if err == nil {
		t.Fatal("run() error = nil, expected the failure to be reported")
	}
	if !strings.Contains(err.Error(), "TopicAuthorizationException") || strings.Contains(err.Error(), "\tat") {
		t.Errorf("run() error = %q, expected the first line of the trace only", err)
	}
}

func TestStateResourceRunTimesOutWhenTheConnectorNeverRuns(t *testing.T) {
	old := runningWaitAttempts
	runningWaitAttempts = 3
	t.Cleanup(func() { runningWaitAttempts = old })
	fake := &fakeConnector{state: "Undefined", startsIn: 100}
	r := newStateResourceAgainst(t, fake)

	err := r.run(context.Background(), "dep1", "Connector")
	if err == nil || !strings.Contains(err.Error(), "did not report itself running") {
		t.Fatalf("run() error = %v, expected a timeout", err)
	}
}

func TestStateResourceStopStopsARunningConnector(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Running"}
	r := newStateResourceAgainst(t, fake)

	if err := r.stop(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	if !equalActions(fake.sent(), "STOP") {
		t.Errorf("actions = %v, expected STOP", fake.sent())
	}
}

func TestStateResourceStopLeavesAStoppedConnectorAlone(t *testing.T) {
	fake := &fakeConnector{state: "Stopped", taskStatus: "Running"}
	r := newStateResourceAgainst(t, fake)

	if err := r.stop(context.Background(), "dep1", "Connector"); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	if len(fake.sent()) != 0 {
		t.Errorf("actions = %v, expected none", fake.sent())
	}
}

// Destroy must leave the deployment UNDEPLOYED: Platform Manager refuses to delete an active
// Connector principal while it is only STOPPED.
func TestStateResourceDeleteStopsAndResets(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  []string
	}{
		{name: "running", state: "Running", want: []string{"STOP", "RESET"}},
		{name: "stopped", state: "Stopped", want: []string{"RESET"}},
		{name: "never started", state: "Undefined", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeConnector{state: test.state, taskStatus: "Running"}
			r := newStateResourceAgainst(t, fake)

			if err := r.resetAfterStop(context.Background(), "dep1"); err != nil {
				t.Fatalf("delete error = %v", err)
			}
			if !equalActions(fake.sent(), test.want...) {
				t.Errorf("actions = %v, expected %v", fake.sent(), test.want)
			}
		})
	}
}

func TestStateResourceApplyRecordsTheLiveStatus(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Running"}
	r := newStateResourceAgainst(t, fake)
	data := applicationDeploymentStateResourceData{
		ApplicationDeployment: types.StringValue("dep1"),
		State:                 types.StringValue(deploymentStateStopped),
	}

	var errs []string
	r.apply(context.Background(), &data, func(summary, detail string) { errs = append(errs, summary+": "+detail) })
	if len(errs) > 0 {
		t.Fatalf("apply() errors = %v", errs)
	}
	if data.CurrentState.ValueString() != "Stopped" {
		t.Errorf("current_state = %q, expected Stopped", data.CurrentState.ValueString())
	}
}

func TestStateResourceApplyReportsAMissingDeployment(t *testing.T) {
	fake := &fakeConnector{gone: true}
	r := newStateResourceAgainst(t, fake)
	data := applicationDeploymentStateResourceData{
		ApplicationDeployment: types.StringValue("dep1"),
		State:                 types.StringValue(deploymentStateRunning),
	}

	var errs []string
	r.apply(context.Background(), &data, func(summary, detail string) { errs = append(errs, summary) })
	if len(errs) != 1 {
		t.Fatalf("apply() errors = %v, expected one", errs)
	}
}

func TestObservedDeploymentState(t *testing.T) {
	tests := []struct {
		name           string
		deploymentType string
		body           string
		expected       string
	}{
		{name: "connector running", deploymentType: "Connector", body: `{"connectorState":{"state":"Running"},"taskStates":[{"id":0,"status":"Running"}]}`, expected: deploymentStateRunning},
		{name: "connector starting", deploymentType: "Connector", body: `{"connectorState":{"state":"Starting"}}`, expected: deploymentStateRunning},
		{name: "connector stopped", deploymentType: "Connector", body: `{"connectorState":{"state":"Stopped"}}`, expected: deploymentStateStopped},
		{name: "connector undefined", deploymentType: "Connector", body: `{"connectorState":{"state":"Undefined"}}`, expected: deploymentStateStopped},
		{name: "connector failed", deploymentType: "Connector", body: `{"connectorState":{"state":"Failed"}}`, expected: deploymentStateFailed},
		{name: "connector running with a failed task", deploymentType: "Connector", body: `{"connectorState":{"state":"Running"},"taskStates":[{"id":0,"status":"Running"},{"id":1,"status":"Failed"}]}`, expected: deploymentStateFailed},
		{name: "connector without state", deploymentType: "Connector", body: `{}`, expected: ""},
		{name: "ksml running", deploymentType: "Ksml", body: `{"ksmlStatus":{"status":"Running"}}`, expected: deploymentStateRunning},
		{name: "ksml undeployed", deploymentType: "Ksml", body: `{"ksmlStatus":{"status":"Undeployed"}}`, expected: deploymentStateStopped},
		{name: "ksml failed", deploymentType: "Ksml", body: `{"ksmlStatus":{"status":"Failed"}}`, expected: deploymentStateFailed},
		{name: "flink running", deploymentType: webclient.FlinkSQLApplicationType, body: `{"flinkStatus":{"status":"Running"}}`, expected: deploymentStateRunning},
		{name: "flink failing", deploymentType: webclient.FlinkSQLApplicationType, body: `{"flinkStatus":{"status":"Failing"}}`, expected: deploymentStateFailed},
		{name: "flink cancelled", deploymentType: webclient.FlinkSQLApplicationType, body: `{"flinkStatus":{"status":"Cancelled"}}`, expected: deploymentStateStopped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var status webclient.ApplicationDeploymentStatusResponse
			if err := json.Unmarshal([]byte(test.body), &status); err != nil {
				t.Fatal(err)
			}
			if got := observedDeploymentState(test.deploymentType, &status); got != test.expected {
				t.Errorf("observedDeploymentState() = %q, expected %q", got, test.expected)
			}
		})
	}
}

func TestLiveDeploymentStatus(t *testing.T) {
	status := webclient.ApplicationDeploymentStatusResponse{}
	status.ConnectorState.State = "Running"
	status.KsmlStatus.Status = "Undeployed"
	status.FlinkStatus.Status = "Failed"
	for deploymentType, expected := range map[string]string{
		"Connector":                       "Running",
		"Ksml":                            "Undeployed",
		webclient.FlinkSQLApplicationType: "Failed",
	} {
		if got := liveDeploymentStatus(deploymentType, &status); got != expected {
			t.Errorf("liveDeploymentStatus(%s) = %q, expected %q", deploymentType, got, expected)
		}
	}
}

func TestFailureTrace(t *testing.T) {
	status := webclient.ApplicationDeploymentStatusResponse{}
	if got := failureTrace(&status); got != "" {
		t.Errorf("failureTrace() = %q, expected empty", got)
	}
	if err := json.Unmarshal([]byte(`{"taskStates":[{"id":0,"status":"Failed","trace":"first line\nsecond line"}]}`), &status); err != nil {
		t.Fatal(err)
	}
	if got := failureTrace(&status); got != ": first line" {
		t.Errorf("failureTrace() = %q, expected the task's first trace line", got)
	}
}

func TestIsAutostart(t *testing.T) {
	for _, test := range []struct {
		value    types.Bool
		expected bool
	}{
		{types.BoolNull(), true},
		{types.BoolUnknown(), true},
		{types.BoolValue(true), true},
		{types.BoolValue(false), false},
	} {
		if got := isAutostart(test.value); got != test.expected {
			t.Errorf("isAutostart(%v) = %t, expected %t", test.value, got, test.expected)
		}
	}
}

func TestIsOnlyAutostartChange(t *testing.T) {
	configs := func(value string) types.Map {
		return types.MapValueMust(types.StringType, map[string]attr.Value{"topics": types.StringValue(value)})
	}
	base := func() ApplicationDeploymentResourceData {
		return ApplicationDeploymentResourceData{
			Configs:           configs("a"),
			Definition:        types.StringNull(),
			DeploymentSize:    types.StringNull(),
			RestartPolicy:     types.StringNull(),
			TargetId:          types.StringValue("cluster"),
			TargetVersion:     types.StringValue("1.0.0"),
			SqlScript:         types.StringNull(),
			GenerateTablesSql: types.BoolNull(),
			Autostart:         types.BoolValue(true),
		}
	}

	tests := []struct {
		name     string
		change   func(plan *ApplicationDeploymentResourceData)
		expected bool
	}{
		{name: "only autostart", change: func(p *ApplicationDeploymentResourceData) { p.Autostart = types.BoolValue(false) }, expected: true},
		{name: "configs", change: func(p *ApplicationDeploymentResourceData) { p.Configs = configs("b") }, expected: false},
		{name: "target", change: func(p *ApplicationDeploymentResourceData) { p.TargetId = types.StringValue("other") }, expected: false},
		{name: "target version", change: func(p *ApplicationDeploymentResourceData) { p.TargetVersion = types.StringValue("2.0.0") }, expected: false},
		{name: "unknown target", change: func(p *ApplicationDeploymentResourceData) { p.TargetId = types.StringUnknown() }, expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, plan := base(), base()
			test.change(&plan)
			if got := isOnlyAutostartChange(&state, &plan); got != test.expected {
				t.Errorf("isOnlyAutostartChange() = %t, expected %t", got, test.expected)
			}
		})
	}
}

func TestMapLiveStatus(t *testing.T) {
	for live, expected := range map[string]string{
		"": "", "Running": deploymentStateRunning, "STARTING": deploymentStateRunning,
		"Stopped": deploymentStateStopped, "Undefined": deploymentStateStopped, "Undeployed": deploymentStateStopped,
		"Failed": deploymentStateFailed, "Failing": deploymentStateFailed,
	} {
		if got := mapLiveStatus(live); got != expected {
			t.Errorf("mapLiveStatus(%q) = %q, expected %q", live, got, expected)
		}
	}
}

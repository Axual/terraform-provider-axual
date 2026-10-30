package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestIsGone(t *testing.T) {
	tests := map[string]struct {
		err  error
		gone bool
	}{
		"404":                      {webclient.NotFoundError, true},
		"grant gone (400)":         {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"No such application access exists with this request!"}`}, true},
		"deployment gone (400)":    {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Required URI template variable 'uid' ... converted to null"}`}, true},
		"group gone (400)":         {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"No Group found"}`}, true},
		"application gone (400)":   {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"No application found"}`}, true},
		"flink cluster gone (400)": {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Flink Cluster with ID f1 not found"}`}, true},
		"other 400":                {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Invalid action"}`}, false},
		"500":                      {&webclient.HTTPError{StatusCode: 500}, false},
		"no error":                 {nil, false},
	}
	for name, test := range tests {
		if got := isGone(test.err); got != test.gone {
			t.Errorf("%s: isGone = %t, expected %t", name, got, test.gone)
		}
	}
}

func TestDeleteRetriesAConflict(t *testing.T) {
	t.Cleanup(shorten(&conflictDelay))
	calls := 0
	err := deleteWithRetry(context.Background(), connectorApplicationType, 0, func() error {
		calls++
		if calls < 3 {
			return &webclient.HTTPError{StatusCode: 409, Body: `{"message":"Could not commit changes!"}`}
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("delete = %v after %d calls, expected success after 3", err, calls)
	}
}

// emptyState is an empty state of r's schema.
func emptyState(t *testing.T, r resource.Resource) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
}

// stateFor is a state of r's schema holding data.
func stateFor(t *testing.T, r resource.Resource, data any) tfsdk.State {
	t.Helper()
	state := emptyState(t, r)
	if diags := state.Set(context.Background(), data); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return state
}

// deleteResource runs Delete of r on a state holding data, against a fake Platform Manager.
func deleteResource(t *testing.T, r resource.Resource, data any) resource.DeleteResponse {
	t.Helper()
	state := stateFor(t, r, data)
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	return resp
}

func fakePM(t *testing.T, handler http.HandlerFunc) *webclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &webclient.Client{HTTPClient: server.Client(), ApiURL: server.URL}
}

// The approval is done when its grant is gone or revoked, also when its own revoke lost the race.
func TestApprovalDeleteAcceptsAGoneOrRevokedGrant(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"grant gone": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"grant gone (400)": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"No such application access exists with this request!"}`))
		},
		"already revoked": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"uid":"g1","status":"Revoked","_links":{}}`))
		},
	}
	var mu sync.Mutex
	revoked := false
	tests["revoked at the same time"] = func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost {
			revoked = true // another resource revoked it first; this request is refused
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Trying to reject or revoke request in invalid state."}`))
			return
		}
		if revoked {
			_, _ = w.Write([]byte(`{"uid":"g1","status":"Revoked","_links":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"uid":"g1","status":"Approved","_links":{"revoke":{"href":"http://pm/revoke"}}}`))
	}
	for name, handler := range tests {
		r := &applicationAccessGrantApprovalResource{provider: AxualProvider{client: fakePM(t, handler)}}
		resp := deleteResource(t, r, &GrantApprovalData{ApplicationAccessGrant: types.StringValue("g1")})
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: delete error = %v", name, resp.Diagnostics)
		}
	}
}

// The deployment state is done when its deployment was deleted at the same time; Platform Manager
// then answers the status read with 400 "... converted to null".
func TestDeploymentStateDeleteAcceptsAGoneDeployment(t *testing.T) {
	var mu sync.Mutex
	reads := 0
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/status") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Required URI template variable 'uid' for method parameter type is present but converted to null"}`))
			return
		}
		reads++
		if reads == 1 {
			_, _ = w.Write([]byte(`{"uid":"d1","_embedded":{"application":{"applicationType":"Connector"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	r := &applicationDeploymentStateResource{provider: AxualProvider{client: client}}
	resp := deleteResource(t, r, &applicationDeploymentStateResourceData{
		Id: types.StringValue("d1"), ApplicationDeployment: types.StringValue("d1"),
		State: types.StringValue("RUNNING"), CurrentState: types.StringValue("Running"),
	})
	if resp.Diagnostics.HasError() {
		t.Errorf("delete error = %v, expected none for a deployment that is gone", resp.Diagnostics)
	}
}

// A STOP refused because the deployment stopped meanwhile is not an error.
func TestStopAfterA4xxIsDoneWhenTheStatusShowsStopped(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Running", stopFails: 1, stopFailsAfterApplying: true, stopFailStatus: http.StatusBadRequest}
	r := newStateResourceAgainst(t, fake)

	if _, err := stopDeploymentAndWait(context.Background(), r.provider.client, "dep1", connectorApplicationType); err != nil {
		t.Fatalf("stop error = %v, expected none", err)
	}
	if !equalActions(fake.sent(), "STOP") {
		t.Errorf("actions = %v, expected one STOP", fake.sent())
	}
}

// The grant is done when it is gone (PM answers 400) or when another resource revoked or removed it
// while this one was revoking it.
func TestGrantDeleteAcceptsAGoneOrRevokedGrant(t *testing.T) {
	gone := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"No such application access exists with this request!"}`))
	}
	tests := map[string]http.HandlerFunc{
		"grant gone (400)": func(w http.ResponseWriter, _ *http.Request) { gone(w) },
	}
	var mu sync.Mutex
	revoked := false
	tests["removed while revoking"] = func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost {
			revoked = true
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Trying to reject or revoke request in invalid state."}`))
			return
		}
		if revoked {
			gone(w)
			return
		}
		_, _ = w.Write([]byte(`{"uid":"g1","status":"Approved","_links":{"revoke":{"href":"http://pm/revoke"}}}`))
	}
	for name, handler := range tests {
		r := &applicationAccessGrantResource{provider: AxualProvider{client: fakePM(t, handler)}}
		resp := deleteResource(t, r, &applicationAccessGrantData{Id: types.StringValue("g1")})
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: delete error = %v", name, resp.Diagnostics)
		}
	}
}

// A STOP refused with "in state STOPPED" is done, also while the worker still reports the connector
// as running.
func TestStopRefusedAsStoppedWhileWorkerStillRuns(t *testing.T) {
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Invalid action STOP for deployment in state STOPPED"}`))
			return
		}
		_, _ = w.Write([]byte(`{"connectorState":{"state":"Running"},"_links":{"stop":{"href":"http://pm/stop"}}}`))
	})
	if err := stopWithRetry(context.Background(), client, "d1", connectorApplicationType); err != nil {
		t.Errorf("stop error = %v, expected none", err)
	}
}

// Ctrl+C cancels the context: no further STOP is sent, and the wait stops at once.
func TestStopRetryStopsWhenCancelled(t *testing.T) {
	saved := stopRetryDelay
	stopRetryDelay = time.Minute
	t.Cleanup(func() { stopRetryDelay = saved })
	var stops int32
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			atomic.AddInt32(&stops, 1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"connectorState":{"state":"Running"},"_links":{"stop":{"href":"http://pm/stop"}}}`))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	err := stopWithRetry(ctx, client, "d1", connectorApplicationType)
	if !errors.Is(err, context.Canceled) || atomic.LoadInt32(&stops) != 1 || time.Since(started) > 5*time.Second {
		t.Errorf("err = %v after %d STOPs in %s, expected context.Canceled after 1 STOP at once", err, stops, time.Since(started))
	}
}

func TestRetryIfStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := RetryIf(ctx, 5, time.Minute, func(error) bool { return true }, func() error {
		calls++
		return errors.New("conflict")
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Errorf("err = %v after %d calls, expected context.Canceled after 1 call", err, calls)
	}
}

// Destroy of a resource that was deleted outside Terraform is done, not an error. A gone group or
// Flink cluster gets a 400, as on Platform Manager master.
func TestDeleteAcceptsAGoneResource(t *testing.T) {
	client := fakePM(t, pmGoneAnswers)
	provider := AxualProvider{client: client}
	flinkCluster := &flinkClusterResource{provider: provider}
	registries := emptyState(t, flinkCluster).Schema.GetAttributes()["schema_registries"].GetType().(types.ListType)
	tests := map[string]struct {
		resource resource.Resource
		data     any
	}{
		"user": {&userResource{provider: provider}, &userResourceData{Id: types.StringValue("u1"), Roles: types.SetNull(roleObjectType)}},
		"group": {&groupResource{provider: provider}, &groupResourceData{
			Id: types.StringValue("g1"), Members: types.SetNull(types.StringType), Managers: types.SetNull(types.StringType),
			StoredMembers: types.SetNull(types.StringType), StoredManagers: types.SetNull(types.StringType),
		}},
		"flink cluster": {flinkCluster, &FlinkClusterResourceData{
			Id: types.StringValue("f1"), InstanceId: types.StringValue("i1"), ClusterId: types.StringValue("c1"),
			SchemaRegistries: types.ListNull(registries.ElemType),
		}},
		"topic browse permissions": {&topicBrowsePermissionsResource{provider: provider}, &topicBrowsePermissionsResourceData{
			TopicConfig: types.StringValue("tc1"),
			Users:       types.SetValueMust(types.StringType, []attr.Value{types.StringValue("u1")}),
			Groups:      types.SetNull(types.StringType),
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if resp := deleteResource(t, test.resource, test.data); resp.Diagnostics.HasError() {
				t.Errorf("delete error = %v, expected none for a resource that is gone", resp.Diagnostics)
			}
		})
	}
}

// pmGoneAnswers answers like Platform Manager master for resources that do not exist.
func pmGoneAnswers(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.Contains(r.URL.Path, "/flink-clusters/"):
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Flink Cluster with ID f1 not found"}`))
	case strings.Contains(r.URL.Path, "/groups/"):
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"No Group found"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// A Flink cluster deleted outside Terraform is removed from the state, so destroy and plan go on.
func TestFlinkClusterReadDropsAGoneCluster(t *testing.T) {
	r := &flinkClusterResource{provider: AxualProvider{client: fakePM(t, pmGoneAnswers)}}
	registries := emptyState(t, r).Schema.GetAttributes()["schema_registries"].GetType().(types.ListType)
	state := stateFor(t, r, &FlinkClusterResourceData{
		Id: types.StringValue("f1"), InstanceId: types.StringValue("i1"), ClusterId: types.StringValue("c1"),
		SchemaRegistries: types.ListNull(registries.ElemType),
	})
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("read = %v, state null = %t; expected the cluster removed from the state", resp.Diagnostics, resp.State.Raw.IsNull())
	}
}

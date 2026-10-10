package provider

import (
	webclient "axual-webclient"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

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
		"404":                   {webclient.NotFoundError, true},
		"grant gone (400)":      {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"No such application access exists with this request!"}`}, true},
		"deployment gone (400)": {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Required URI template variable 'uid' ... converted to null"}`}, true},
		"other 400":             {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Invalid action"}`}, false},
		"500":                   {&webclient.HTTPError{StatusCode: 500}, false},
		"no error":              {nil, false},
	}
	for name, test := range tests {
		if got := isGone(test.err); got != test.gone {
			t.Errorf("%s: isGone = %t, expected %t", name, got, test.gone)
		}
	}
}

func TestIsConflict(t *testing.T) {
	if !isConflict(&webclient.HTTPError{StatusCode: 409, Body: `{"message":"Could not commit changes!"}`}) {
		t.Error("409 is a conflict")
	}
	if !isConflict(&webclient.HTTPError{StatusCode: 403, Body: "org.springframework.orm.ObjectOptimisticLockingFailureException: ..."}) {
		t.Error("403 optimistic lock is a conflict")
	}
	if isConflict(&webclient.HTTPError{StatusCode: 403, Body: "Forbidden"}) {
		t.Error("a plain 403 is not a conflict")
	}
}

func TestDeleteRetriesAConflict(t *testing.T) {
	t.Cleanup(shorten(&conflictDelay))
	calls := 0
	err := deleteWithRetry(connectorApplicationType, 0, func() error {
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

// deleteResource runs Delete of r on a state holding data, against a fake Platform Manager.
func deleteResource(t *testing.T, r resource.Resource, data any) resource.DeleteResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, data); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	resp := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
	return resp
}

func fakePM(t *testing.T, handler http.HandlerFunc) *webclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &webclient.Client{HTTPClient: server.Client(), ApiURL: server.URL}
}

// A destroy of generated config (no references) revokes the grant and its approval at the same time:
// the approval is done when the grant is gone or already revoked, also when its own revoke lost the race.
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
func TestStopRefusedBecauseAlreadyStoppedIsFine(t *testing.T) {
	fake := &fakeConnector{state: "Running", taskStatus: "Running", stopFails: 1, stopFailsAfterApplying: true, stopFailStatus: http.StatusBadRequest}
	r := newStateResourceAgainst(t, fake)

	if err := stopDeploymentAndWait(context.Background(), r.provider.client, "dep1", connectorApplicationType); err != nil {
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

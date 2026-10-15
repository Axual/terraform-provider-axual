package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

// statusServer answers the status read with the given codes, one per request, then 200 Running.
func statusServer(t *testing.T, codes ...int) (*webclient.Client, *int32) {
	t.Helper()
	var calls int32
	client := fakePM(t, func(w http.ResponseWriter, _ *http.Request) {
		n := int(atomic.AddInt32(&calls, 1))
		if n <= len(codes) {
			w.WriteHeader(codes[n-1])
			_, _ = w.Write([]byte(`{"detail":"None of the servers could handle the get Connector status request. Status Code: 500"}`))
			return
		}
		_, _ = w.Write([]byte(`{"connectorState":{"state":"Running"},"_links":{"stop":{"href":"http://pm/stop"}}}`))
	})
	return client, &calls
}

// Platform Manager answers 500 for about a minute after the Connect workers restart.
func TestStatusReadRetriesA5xx(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	client, calls := statusServer(t, http.StatusInternalServerError, http.StatusBadGateway)

	status, err := getDeploymentStatus(context.Background(), client, "d1")
	if err != nil || status.ConnectorState.State != "Running" || *calls != 3 {
		t.Errorf("status = %v, err = %v after %d calls, expected Running after 3", status, err, *calls)
	}
}

func TestStatusReadDoesNotRetryA4xx(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	client, calls := statusServer(t, http.StatusBadRequest)

	if _, err := getDeploymentStatus(context.Background(), client, "d1"); err == nil || *calls != 1 {
		t.Errorf("err = %v after %d calls, expected the 400 at once", err, *calls)
	}
}

// A 422 is a 4xx too, although the webclient does not return it as an HTTPError.
func TestStatusReadDoesNotRetryA422(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	client, calls := statusServer(t, http.StatusUnprocessableEntity)

	if _, err := getDeploymentStatus(context.Background(), client, "d1"); err == nil || *calls != 1 {
		t.Errorf("err = %v after %d calls, expected the 422 at once", err, *calls)
	}
}

func TestIsTransientError(t *testing.T) {
	refusedToken := &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}}
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "5xx", err: &webclient.HTTPError{StatusCode: http.StatusInternalServerError}, expected: true},
		{name: "no answer", err: &url.Error{Op: "Get", URL: "https://pm", Err: errors.New("connection refused")}, expected: true},
		{name: "404", err: webclient.NotFoundError, expected: false},
		{name: "422", err: webclient.UnprocessableEntityError, expected: false},
		{name: "4xx", err: &webclient.HTTPError{StatusCode: http.StatusBadRequest}, expected: false},
		{name: "unreadable body", err: &json.SyntaxError{}, expected: false},
		{name: "refused token", err: &url.Error{Op: "Get", URL: "https://pm", Err: fmt.Errorf("renewing the access token failed: %w", refusedToken)}, expected: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTransientError(test.err); got != test.expected {
				t.Errorf("isTransientError() = %t, expected %t", got, test.expected)
			}
		})
	}
}

// Ctrl+C in Terraform cancels the context: the wait between attempts stops at once.
func TestStatusReadStopsWhenCancelled(t *testing.T) {
	saved := statusRetryDelay
	statusRetryDelay = time.Minute
	t.Cleanup(func() { statusRetryDelay = saved })
	client, calls := statusServer(t, http.StatusInternalServerError, http.StatusInternalServerError)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	_, err := getDeploymentStatus(ctx, client, "d1")
	if !errors.Is(err, context.Canceled) || *calls != 1 || time.Since(started) > 5*time.Second {
		t.Errorf("err = %v after %d calls in %s, expected context.Canceled after 1 call at once", err, *calls, time.Since(started))
	}
}

func TestStatusReadGivesUp(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	codes := make([]int, statusAttempts)
	for i := range codes {
		codes[i] = http.StatusInternalServerError
	}
	client, calls := statusServer(t, codes...)

	_, err := getDeploymentStatus(context.Background(), client, "d1")
	var httpErr *webclient.HTTPError
	if !errors.As(err, &httpErr) || int(*calls) != statusAttempts {
		t.Errorf("err = %v after %d calls, expected the 500 after %d", err, *calls, statusAttempts)
	}
}

// A plan right after the Connect workers restart: the deployment state is read through the 500.
func TestDeploymentStateReadRidesOutA500(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	var statusCalls int32
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			if atomic.AddInt32(&statusCalls, 1) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"connectorState":{"state":"Running"},"taskStates":[{"id":0,"status":"Running"}],"_links":{"stop":{"href":"http://pm/stop"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"uid":"d1","_embedded":{"application":{"applicationType":"Connector"}}}`))
	})
	r := &applicationDeploymentStateResource{provider: AxualProvider{client: client}}
	ctx := context.Background()
	state := stateFor(t, r, &applicationDeploymentStateResourceData{
		Id: types.StringValue("d1"), ApplicationDeployment: types.StringValue("d1"),
		State: types.StringValue("RUNNING"), CurrentState: types.StringValue("Running"),
	})
	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)

	var got applicationDeploymentStateResourceData
	resp.State.Get(ctx, &got)
	if resp.Diagnostics.HasError() || got.State.ValueString() != "RUNNING" || statusCalls != 2 {
		t.Errorf("read = %v, state %s after %d status reads, expected RUNNING after 2", resp.Diagnostics, got.State, statusCalls)
	}
}

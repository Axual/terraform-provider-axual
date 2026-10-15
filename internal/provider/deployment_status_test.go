package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
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
		{name: "connection refused", err: &url.Error{Op: "Get", URL: "https://pm", Err: &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}}, expected: true},
		{name: "connection dropped", err: &url.Error{Op: "Get", URL: "https://pm", Err: io.EOF}, expected: true},
		{name: "timeout", err: &url.Error{Op: "Get", URL: "https://pm", Err: context.DeadlineExceeded}, expected: true},
		{name: "unknown host", err: &url.Error{Op: "Get", URL: "https://pm", Err: &net.DNSError{Err: "no such host", Name: "pm", IsNotFound: true}}, expected: false},
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

// Errors that a retry cannot fix fail at once, also when they come from the network.
func TestStatusReadDoesNotRetryABadCertificateOrURL(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	var calls int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { atomic.AddInt32(&calls, 1) }))
	t.Cleanup(server.Close)
	tests := map[string]*webclient.Client{
		"untrusted certificate":       {HTTPClient: &http.Client{}, ApiURL: server.URL},
		"unsupported protocol scheme": {HTTPClient: server.Client(), ApiURL: "htps://pm"},
	}
	for name, client := range tests {
		t.Run(name, func(t *testing.T) {
			started := time.Now()
			_, err := getDeploymentStatus(context.Background(), client, "d1")
			if err == nil || strings.Contains(err.Error(), "attempts") || time.Since(started) > time.Second {
				t.Errorf("err = %v in %s, expected the first error at once", err, time.Since(started))
			}
		})
	}
	if calls != 0 {
		t.Errorf("the handler was called %d times, expected 0", calls)
	}
}

// A real connection refused is retried, because Platform Manager can be starting.
func TestStatusReadRetriesARefusedConnection(t *testing.T) {
	t.Cleanup(shorten(&statusRetryDelay))
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	client := &webclient.Client{HTTPClient: &http.Client{}, ApiURL: server.URL}

	_, err := getDeploymentStatus(context.Background(), client, "d1")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("after %d attempts", statusAttempts)) {
		t.Errorf("err = %v, expected it after %d attempts", err, statusAttempts)
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

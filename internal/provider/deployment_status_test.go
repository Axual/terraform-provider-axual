package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	state.Set(ctx, &applicationDeploymentStateResourceData{
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

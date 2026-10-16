package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// readDeploymentAgainst reads a deployment from a fake Platform Manager that answers every request with handler.
func readDeploymentAgainst(t *testing.T, handler http.HandlerFunc) resource.ReadResponse {
	t.Helper()
	r := &applicationDeploymentResource{provider: AxualProvider{client: fakePM(t, handler)}}
	state := stateFor(t, r, &ApplicationDeploymentResourceData{
		Id:          types.StringValue("dep1"),
		Application: types.StringValue("app1"),
		Environment: types.StringValue("env1"),
		Configs:     types.MapNull(types.StringType),
	})

	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

// A deployment deleted outside Terraform is removed from the state, so the next plan creates it again.
func TestDeploymentReadRemovesAGoneDeployment(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "empty search result", handler: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"_embedded":{"application_deployments":[]}}`))
		}},
		{name: "404", handler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := readDeploymentAgainst(t, test.handler)
			if resp.Diagnostics.HasError() {
				t.Fatalf("read error = %v", resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("state is kept, expected the deployment to be removed from the state")
			}
		})
	}
}

// Any other error is reported, and the deployment stays in the state.
func TestDeploymentReadReportsOtherErrors(t *testing.T) {
	resp := readDeploymentAgainst(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if !resp.Diagnostics.HasError() {
		t.Error("no error, expected the 500 to be reported")
	}
	if resp.State.Raw.IsNull() {
		t.Error("state is removed, expected it to be kept")
	}
}

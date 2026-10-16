package provider

import (
	webclient "axual-webclient"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeAPI answers GET path (without query) with a status and body.
func fakeAPI(t *testing.T, routes map[string]struct {
	status int
	body   string
}) *webclient.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(route.status)
		_, _ = w.Write([]byte(route.body))
	}))
	t.Cleanup(srv.Close)
	return &webclient.Client{HTTPClient: srv.Client(), ApiURL: srv.URL}
}

func TestResolveEnvironmentExactName(t *testing.T) {
	c := fakeAPI(t, map[string]struct {
		status int
		body   string
	}{
		"/environments/dev": {http.StatusBadRequest, `{"detail":"not an id"}`},
		// An API that matches part of the value returns dev2 too.
		"/environments/search/findByShortName": {http.StatusOK, `{"_embedded":{"environments":[{"uid":"a","shortName":"dev","name":"Development"},{"uid":"b","shortName":"dev2","name":"Dev 2"}]}}`},
		"/environments/search/findByName":      {http.StatusOK, `{"_embedded":{"environments":[]}}`},
	})
	if id, err := resolveEnvironment(c, "dev"); err != nil || id != "a" {
		t.Errorf("got %q, %v; want a", id, err)
	}
}

func TestResolveGroupExactName(t *testing.T) {
	c := fakeAPI(t, map[string]struct {
		status int
		body   string
	}{
		"/groups/search/findByName": {http.StatusOK, `{"_embedded":{"groups":[{"uid":"x","name":"Team A"}]}}`},
	})
	if _, err := resolveGroup(c, "Team"); err == nil || !strings.Contains(err.Error(), `no group "Team" found`) {
		t.Errorf("a partial name must not match: got %v", err)
	}
	if id, err := resolveGroup(c, "Team A"); err != nil || id != "x" {
		t.Errorf("got %q, %v; want x", id, err)
	}
}

func TestResolveGroupKeepsRealErrors(t *testing.T) {
	c := fakeAPI(t, map[string]struct {
		status int
		body   string
	}{
		"/groups/Team": {http.StatusInternalServerError, `{"detail":"boom"}`},
	})
	if _, err := resolveGroup(c, "Team"); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("a 500 must be returned, not hidden as 'not found': got %v", err)
	}
}

// An application is found by its short name or by its exact name.
func TestResolveApplicationByShortOrFullName(t *testing.T) {
	c := fakeAPI(t, map[string]struct {
		status int
		body   string
	}{
		"/applications/search/findByShortName": {http.StatusOK, `{"uid":"a","shortName":"orders","name":"Orders service"}`},
		"/applications/search/findByName":      {http.StatusOK, `{"uid":"a","shortName":"orders","name":"Orders service"}`},
	})
	for _, v := range []string{"orders", "Orders service"} {
		if id, err := resolveApplication(c, v); err != nil || id != "a" {
			t.Errorf("%q: got %q, %v; want a", v, id, err)
		}
	}
	if _, err := resolveApplication(c, "Orders"); err == nil || !strings.Contains(err.Error(), `no application "Orders" found`) {
		t.Errorf("a name in another case must not match: got %v", err)
	}
}

// The filters with a fixed set of values refuse others at plan time, in any case.
func TestListFilterValues(t *testing.T) {
	ctx := context.Background()
	schemaOf := func(typeName string) listschema.Schema {
		for _, s := range listSpecs() {
			if s.typeName == typeName {
				var resp list.ListResourceSchemaResponse
				newListResource(AxualProvider{}, s).ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, &resp)
				return resp.Schema
			}
		}
		t.Fatalf("no list resource %s", typeName)
		return listschema.Schema{}
	}
	valid := func(s listschema.Schema, attr, value string) bool {
		var diags diag.Diagnostics
		for _, v := range s.Attributes[attr].(listschema.StringAttribute).Validators {
			resp := validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root(attr), ConfigValue: types.StringValue(value)}, &resp)
			diags.Append(resp.Diagnostics...)
		}
		return !diags.HasError()
	}
	apps := schemaOf("axual_application")
	for value, want := range map[string]bool{"Ksml": true, "flink_sql": true, "Custom": true, "Flink": false, "KSQL": false} {
		if got := valid(apps, "application_type", value); got != want {
			t.Errorf("application_type %q valid = %v, expected %v", value, got, want)
		}
	}
	grants := schemaOf("axual_application_access_grant")
	if !valid(grants, "access_type", "producer") || valid(grants, "access_type", "READ") || !valid(grants, "status", "revoked") {
		t.Error("access_type and status must accept their values in any case and refuse others")
	}
}

// Without a status filter, revoked and cancelled grants are left out; a status filter or an
// approval list asks for exactly that status.
func TestGrantListStatuses(t *testing.T) {
	var got []string
	c := &webclient.Client{ApiURL: ""}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()["statuses"]
		_, _ = w.Write([]byte(`{"_embedded":{"application_access_grants":[]},"page":{"totalPages":1}}`))
	}))
	t.Cleanup(srv.Close)
	c.HTTPClient, c.ApiURL = srv.Client(), srv.URL
	ctx := context.Background()
	tests := []struct {
		filters map[string]string
		status  string
		want    string
	}{
		{map[string]string{}, "", "PENDING,APPROVED,REJECTED"},
		{map[string]string{"status": "revoked"}, "", "REVOKED"},
		{map[string]string{}, "APPROVED", "APPROVED"},
	}
	for _, test := range tests {
		if _, err := findGrants(ctx, c, test.filters, test.status, false); err != nil || strings.Join(got, ",") != test.want {
			t.Errorf("filters %v, status %q: sent statuses %v (err %v), expected %s", test.filters, test.status, got, err, test.want)
		}
	}
}

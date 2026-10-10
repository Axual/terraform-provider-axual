package provider

import (
	webclient "axual-webclient"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveOne(t *testing.T) {
	items := []webclient.ListItem{
		{"uid": "a", "name": "orders"},
		{"uid": "b", "name": "orders-archive"},
	}
	exact := func(it webclient.ListItem) bool { return it.String("name") == "orders" }

	if id, err := resolveOne("topic", "orders", items, exact); err != nil || id != "a" {
		t.Errorf("one match: got %q, %v; want a", id, err)
	}
	if _, err := resolveOne("topic", "orders", items, nil); err == nil || !strings.Contains(err.Error(), "more than one topic") {
		t.Errorf("two matches: got %v, want an error about more than one", err)
	}
	if _, err := resolveOne("topic", "payments", nil, nil); err == nil || !strings.Contains(err.Error(), `no topic found with ID or name "payments"`) {
		t.Errorf("no match: got %v, want a not-found error", err)
	}
}

func TestSameJSON(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{`{"type":"record","name":"A"}`, "{\n  \"name\": \"A\",\n  \"type\": \"record\"\n}", true},
		{`{"name":"A"}`, `{"name":"B"}`, false},
		{`syntax = "proto3";`, `syntax = "proto3";`, false}, // not JSON
	}
	for _, tt := range tests {
		if got := sameJSON(tt.a, tt.b); got != tt.want {
			t.Errorf("sameJSON(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

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
	if _, err := resolveGroup(c, "Team"); err == nil || !strings.Contains(err.Error(), `no group found with ID or name "Team"`) {
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

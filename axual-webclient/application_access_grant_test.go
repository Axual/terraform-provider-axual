package webclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const grantSearchResponse = `{"_embedded":{"applicationAccessGrantResponses":[{"uid":"g1","status":"APPROVED","accessType":"CONSUMER",` +
	`"application":{"uid":"a1","shortName":"payapp"},"stream":{"uid":"s1","name":"payments"},` +
	`"environment":{"uid":"e1","shortName":"dev","autoApproved":true}}]},"page":{"totalPages":1}}`

// oldGrantSearchResponse is the same grant as findByAttributes answers it, before Platform Manager 17.0.0.
const oldGrantSearchResponse = `{"_embedded":{"applicationAccessGrantResponses":[{"uid":"g1","status":"APPROVED","accessType":"CONSUMER",` +
	`"_embedded":{"application":{"uid":"a1","shortName":"payapp"},"stream":{"uid":"s1","name":"payments"},` +
	`"environment":{"uid":"e1","shortName":"dev"}}}]},"page":{"totalPages":1}}`

const (
	newSearch = "/application_access_grants/search/findSummariesByAttributes"
	oldSearch = "/application_access_grants/search/findByAttributes"
)

// grantServer answers the grant search with one grant and checks the applicationId filter.
// With old set, it is a Platform Manager before 17.0.0: findSummariesByAttributes answers 404.
// It counts the requests per path in calls.
func grantServer(t *testing.T, old bool, calls map[string]int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		if got := r.URL.Query().Get("applicationId"); got != "a1" {
			t.Errorf("applicationId = %q, want the filter to be passed on", got)
		}
		switch {
		case r.URL.Path == newSearch && !old:
			_, _ = w.Write([]byte(grantSearchResponse))
		case r.URL.Path == oldSearch && old:
			_, _ = w.Write([]byte(oldGrantSearchResponse))
		default:
			http.NotFound(w, r)
		}
	}))
}

func checkGrant(t *testing.T, got *GetApplicationAccessGrantsByAttributeResponse) {
	t.Helper()
	grants := got.Embedded.ApplicationAccessGrantResponses
	if len(grants) != 1 {
		t.Fatalf("got %d grants, want 1", len(grants))
	}
	g := grants[0]
	if g.Uid != "g1" || g.Status != "APPROVED" || g.AccessType != "CONSUMER" ||
		g.Application.Uid != "a1" || g.Application.ShortName != "payapp" ||
		g.Stream.Uid != "s1" || g.Stream.Name != "payments" ||
		g.Environment.Uid != "e1" || g.Environment.ShortName != "dev" {
		t.Errorf("grant = %+v", g)
	}
}

func checkListedGrants(t *testing.T, items []ListItem) {
	t.Helper()
	if len(items) != 1 {
		t.Fatalf("got %d grants, want 1", len(items))
	}
	it := items[0]
	if got := it.Embedded("application", "shortName") + " " + it.Embedded("stream", "name") + " " +
		it.Embedded("environment", "shortName"); got != "payapp payments dev" {
		t.Errorf("name parts = %q, want payapp payments dev", got)
	}
}

func TestGetApplicationAccessGrantsByAttributes(t *testing.T) {
	calls := map[string]int{}
	srv := grantServer(t, false, calls)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	got, err := client.GetApplicationAccessGrantsByAttributes(ApplicationAccessGrantAttributes{ApplicationId: "a1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkGrant(t, got)
	if calls[oldSearch] != 0 {
		t.Errorf("findByAttributes called %d times, want 0", calls[oldSearch])
	}
}

func TestListApplicationAccessGrants(t *testing.T) {
	calls := map[string]int{}
	srv := grantServer(t, false, calls)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	items, err := client.ListApplicationAccessGrants(url.Values{"applicationId": {"a1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkListedGrants(t, items)
	if calls[oldSearch] != 0 {
		t.Errorf("findByAttributes called %d times, want 0", calls[oldSearch])
	}
}

// Before Platform Manager 17.0.0, the search falls back to findByAttributes and reads its _embedded
// shape. After the first 404 it no longer tries findSummariesByAttributes.
func TestGetApplicationAccessGrantsByAttributesBefore17(t *testing.T) {
	calls := map[string]int{}
	srv := grantServer(t, true, calls)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	for range 2 {
		got, err := client.GetApplicationAccessGrantsByAttributes(ApplicationAccessGrantAttributes{ApplicationId: "a1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		checkGrant(t, got)
	}
	if calls[newSearch] != 1 || calls[oldSearch] != 2 {
		t.Errorf("calls = %v, want findSummariesByAttributes once and findByAttributes twice", calls)
	}
}

func TestListApplicationAccessGrantsBefore17(t *testing.T) {
	calls := map[string]int{}
	srv := grantServer(t, true, calls)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	items, err := client.ListApplicationAccessGrants(url.Values{"applicationId": {"a1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkListedGrants(t, items)
	if calls[newSearch] != 1 || calls[oldSearch] != 1 {
		t.Errorf("calls = %v, want each search once", calls)
	}
}

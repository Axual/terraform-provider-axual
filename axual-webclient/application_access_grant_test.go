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

// grantServer answers findSummariesByAttributes with one grant and checks the applicationId filter.
func grantServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/application_access_grants/search/findSummariesByAttributes" {
			t.Errorf("path = %q, want findSummariesByAttributes", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("applicationId"); got != "a1" {
			t.Errorf("applicationId = %q, want the filter to be passed on", got)
		}
		_, _ = w.Write([]byte(grantSearchResponse))
	}))
}

func TestGetApplicationAccessGrantsByAttributes(t *testing.T) {
	srv := grantServer(t)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	got, err := client.GetApplicationAccessGrantsByAttributes(ApplicationAccessGrantAttributes{ApplicationId: "a1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestListApplicationAccessGrants(t *testing.T) {
	srv := grantServer(t)
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	items, err := client.ListApplicationAccessGrants(url.Values{"applicationId": {"a1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d grants, want 1", len(items))
	}
	it := items[0]
	if got := it.Embedded("application", "shortName") + " " + it.Embedded("stream", "name") + " " +
		it.Embedded("environment", "shortName"); got != "payapp payments dev" {
		t.Errorf("name parts = %q, want them read from the plain fields", got)
	}
}

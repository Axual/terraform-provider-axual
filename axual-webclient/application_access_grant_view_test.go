package webclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGrantViewFromItem(t *testing.T) {
	tests := []struct {
		desc string
		body string
		want GrantView
	}{
		{
			desc: "GET response: snake_case dates, processedBy as text",
			body: `{"uid":"g1","status":"Revoked","accessType":"Consumer","comment":"Stopped, INC-1",
				"requested_at":"2026-10-26T09:45:45","processed_at":"2026-10-26T09:45:43","processedBy":"tom@axual.com","requestedBy":"alex@axual.com",
				"_embedded":{"application":{"uid":"a1","name":"Log Scraper"},"stream":{"uid":"t1","name":"logs"},"environment":{"uid":"e1","name":"dev"}},
				"_links":{"self":{"href":"x"}}}`,
			want: GrantView{Uid: "g1", Status: "Revoked", AccessType: "CONSUMER", ApplicationUid: "a1", ApplicationName: "Log Scraper",
				TopicUid: "t1", TopicName: "logs", EnvironmentUid: "e1", EnvironmentName: "dev", Comment: "Stopped, INC-1",
				RequestedAt: "2026-10-26T09:45:45", ProcessedAt: "2026-10-26T09:45:43", ProcessedBy: "tom@axual.com", RequestedBy: "alex@axual.com"},
		},
		{
			desc: "findSummariesByAttributes: plain application, stream and environment, emails as text",
			body: `{"uid":"g3","status":"Approved","accessType":"Consumer","comment":"ok","requestedAt":"2026-10-26T10:00:00","processedAt":"2026-10-26T11:00:00",
				"requestedBy":"alex@axual.com","processedBy":"tom@axual.com","requestedByUser":{"uid":"u2","firstName":"Alex"},"processedByUser":null,
				"application":{"uid":"a3","name":"Billing"},"stream":{"uid":"t3","name":"invoices"},"environment":{"uid":"e3","name":"prod"},
				"_links":{"revoke":{"href":"x"}}}`,
			want: GrantView{Uid: "g3", Status: "Approved", AccessType: "CONSUMER", ApplicationUid: "a3", ApplicationName: "Billing",
				TopicUid: "t3", TopicName: "invoices", EnvironmentUid: "e3", EnvironmentName: "prod", Comment: "ok",
				RequestedAt: "2026-10-26T10:00:00", ProcessedAt: "2026-10-26T11:00:00", ProcessedBy: "tom@axual.com", RequestedBy: "alex@axual.com", CanRevoke: true},
		},
		{
			desc: "findByAttributes: camelCase dates, user objects with emailAddress.email, links",
			body: `{"uid":"g2","status":"Pending","accessType":"Producer","requestedAt":"2026-10-26T10:00:00","processedAt":null,
				"processedByUser":{"uid":"u1","emailAddress":{"email":"tom@axual.com"}},"requestedByUser":{"uid":"u2","emailAddress":{"email":"alex@axual.com"}},
				"_embedded":{"application":{"uid":"a2","name":"POC"}},
				"_links":{"approve":{"href":"x"},"deny":{"href":"y"}}}`,
			want: GrantView{Uid: "g2", Status: "Pending", AccessType: "PRODUCER", ApplicationUid: "a2", ApplicationName: "POC",
				RequestedAt: "2026-10-26T10:00:00", ProcessedBy: "tom@axual.com", RequestedBy: "alex@axual.com", CanApprove: true, CanDeny: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var item ListItem
			if err := json.Unmarshal([]byte(tt.body), &item); err != nil {
				t.Fatal(err)
			}
			if got := grantViewFromItem(item); got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// SearchGrants uses findSummariesByAttributes, and findByAttributes before Platform Manager 17.0.0.
// Both give the emails of the requester and the processor.
func TestSearchGrants(t *testing.T) {
	const newBody = `{"_embedded":{"applicationAccessGrantResponses":[{"uid":"g1","status":"Approved","accessType":"Consumer",` +
		`"requestedBy":"alex@axual.com","processedBy":"tom@axual.com","application":{"uid":"a1","name":"Billing"}}]},"page":{"totalPages":1}}`
	const oldBody = `{"_embedded":{"applicationAccessGrantResponses":[{"uid":"g1","status":"Approved","accessType":"Consumer",` +
		`"requestedByUser":{"emailAddress":{"email":"alex@axual.com"}},"processedByUser":{"emailAddress":{"email":"tom@axual.com"}},` +
		`"_embedded":{"application":{"uid":"a1","name":"Billing"}}}]},"page":{"totalPages":1}}`
	for _, old := range []bool{false, true} {
		calls := map[string]int{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls[r.URL.Path]++
			if got := r.URL.Query().Get("streamId"); got != "t1" {
				t.Errorf("streamId = %q, want the filter to be passed on", got)
			}
			switch {
			case r.URL.Path == newSearch && !old:
				_, _ = w.Write([]byte(newBody))
			case r.URL.Path == oldSearch && old:
				_, _ = w.Write([]byte(oldBody))
			default:
				http.NotFound(w, r)
			}
		}))
		client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}
		got, err := client.SearchGrants("t1", "", "", "", nil)
		srv.Close()
		if err != nil {
			t.Fatalf("old=%v: unexpected error: %v", old, err)
		}
		want := GrantView{Uid: "g1", Status: "Approved", AccessType: "CONSUMER", ApplicationUid: "a1", ApplicationName: "Billing",
			RequestedBy: "alex@axual.com", ProcessedBy: "tom@axual.com"}
		if len(got) != 1 || got[0] != want {
			t.Errorf("old=%v: got %+v, want [%+v]", old, got, want)
		}
		wantOld := 0
		if old {
			wantOld = 1
		}
		if calls[newSearch] != 1 || calls[oldSearch] != wantOld {
			t.Errorf("old=%v: calls = %v", old, calls)
		}
	}
}

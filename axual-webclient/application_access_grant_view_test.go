package webclient

import (
	"encoding/json"
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
			desc: "search response: camelCase dates, user objects with emailAddress.email, links",
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

package provider

import (
	webclient "axual-webclient"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMatchGrant(t *testing.T) {
	grants := []webclient.GrantView{
		{Uid: "g1", ApplicationUid: "0123456789abcdef0123456789abcdef", ApplicationName: "Log Scraper"},
		// A name that looks like the UID of another application.
		{Uid: "g2", ApplicationUid: "ffffffffffffffffffffffffffffffff", ApplicationName: "0123456789abcdef0123456789abcdef"},
	}
	tests := []struct{ application, want string }{
		{"Log Scraper", "g1"},
		{"0123456789abcdef0123456789abcdef", "g1"}, // UID first
		{"ffffffffffffffffffffffffffffffff", "g2"},
		{"Unknown", ""},
	}
	for _, tt := range tests {
		got := ""
		if g := matchGrant(grants, tt.application, ""); g != nil {
			got = g.Uid
		}
		if got != tt.want {
			t.Errorf("matchGrant(%q) = %q, want %q", tt.application, got, tt.want)
		}
	}
}

func TestCompositeDecisionID(t *testing.T) {
	d := grantDecisionData{Topic: types.StringValue("t"), Environment: types.StringValue("e"),
		AccessType: types.StringValue("CONSUMER"), Application: types.StringValue("team/app")}
	if got := compositeDecisionID(d); got != "t/e/CONSUMER/team/app" {
		t.Errorf("got %q", got)
	}
}

func TestVersionBefore(t *testing.T) {
	tests := map[string]bool{"15.0.1": true, "15.0.3": true, "15.0.4": false, "15.1.0": false, "16.0.0": false, "16.1.0-SNAPSHOT": false, "": false, "dev": false}
	for v, want := range tests {
		if got := versionBefore(v, 15, 0, 4); got != want {
			t.Errorf("versionBefore(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestMatchGrantByGrantIDFirst(t *testing.T) {
	grants := []webclient.GrantView{{Uid: "g1", ApplicationName: "Log Scraper Renamed"}, {Uid: "g2", ApplicationName: "Log Scraper"}}
	if g := matchGrant(grants, "Log Scraper", "g1"); g == nil || g.Uid != "g1" {
		t.Errorf("a renamed application must keep its grant, got %+v", g)
	}
}

func TestClosedByRequester(t *testing.T) {
	tests := []struct {
		desc                       string
		status, processed, req, me string
		want                       bool
	}{
		{"application team revoked or cancelled and asked again", "Pending", "alex@x", "alex@x", "tom@x", true},
		{"topic team revoked", "Pending", "tom@x", "alex@x", "tom@x", false},
		{"one account for both sides", "Pending", "ci@x", "ci@x", "ci@x", false},
		{"first request, never decided", "Pending", "", "alex@x", "tom@x", false},
		{"own identity unknown", "Pending", "alex@x", "alex@x", "", false},
		{"not pending", "Approved", "alex@x", "alex@x", "tom@x", false},
	}
	for _, tt := range tests {
		if got := closedByRequester(tt.status, tt.processed, tt.req, tt.me); got != tt.want {
			t.Errorf("%s: got %v", tt.desc, got)
		}
	}
}

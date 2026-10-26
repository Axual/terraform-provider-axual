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
		if g := matchGrant(grants, tt.application); g != nil {
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

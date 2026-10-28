package provider

import (
	webclient "axual-webclient"
	"sync"
	"testing"
	"time"

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

// TestWaiting: decisions planned together share one warning; one planned later starts a new group.
func TestWaiting(t *testing.T) {
	quiet, max := waitQuiet, waitMax
	waitQuiet, waitMax = 40*time.Millisecond, time.Second
	defer func() { waitQuiet, waitMax = quiet, max }()

	c := newGrantCache(nil)
	var mu sync.Mutex
	var groups [][]string
	var wg sync.WaitGroup
	for i, name := range []string{"C", "A", "B"} {
		wg.Add(1)
		go func(delay time.Duration, name string) {
			defer wg.Done()
			time.Sleep(delay)
			if names := c.waiting("logs/dev/CONSUMER", name); names != nil {
				mu.Lock()
				groups = append(groups, names)
				mu.Unlock()
			}
		}(time.Duration(i)*10*time.Millisecond, name)
	}
	wg.Wait()
	if len(groups) != 1 || len(groups[0]) != 3 || groups[0][0] != "A" || groups[0][2] != "C" {
		t.Fatalf("got groups %v, want one group [A B C]", groups)
	}
	if names := c.waiting("logs/dev/CONSUMER", "D"); len(names) != 1 || names[0] != "D" {
		t.Errorf("a later decision starts a new group, got %v", names)
	}
}

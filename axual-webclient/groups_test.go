package webclient

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// memberProbeServer is a fake API that answers GET /service-accounts/{uid} from a set of
// known service account uids, and records every uid it was asked about.
//
// status is what it answers for a uid that is not in that set. 404 is what a real
// Platform Manager says for a person, for an unknown uid, and — since the route does not
// exist there at all — for every uid on a Platform Manager without service accounts.
func memberProbeServer(t *testing.T, status int, serviceAccounts ...string) (*Client, func() []string) {
	t.Helper()

	known := map[string]bool{}
	for _, uid := range serviceAccounts {
		known[uid] = true
	}
	var asked []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := strings.CutPrefix(r.URL.Path, "/service-accounts/")
		if !ok {
			t.Errorf("unexpected request path %q", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		asked = append(asked, uid)
		if known[uid] {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	return &Client{HTTPClient: srv.Client(), ApiURL: srv.URL, Realm: "tenant"},
		func() []string { return asked }
}

func TestGroupMemberURI(t *testing.T) {
	const sa, person = "sa-uid", "person-uid"

	tests := []struct {
		name string
		uid  string
		want string
	}{
		{"a service account is named in its own collection", sa, "/service-accounts/" + sa},
		{"a person is named in the users collection", person, "/users/" + person},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := memberProbeServer(t, http.StatusNotFound, sa)

			got, err := client.GroupMemberURI(tt.uid)
			if err != nil {
				t.Fatalf("GroupMemberURI(%q): unexpected error: %v", tt.uid, err)
			}
			if want := client.ApiURL + tt.want; got != want {
				t.Errorf("GroupMemberURI(%q) = %q, want %q", tt.uid, got, want)
			}
		})
	}
}

// A mixed member list is the case this whole mechanism exists for: one write carrying
// both kinds, each needing a different collection.
func TestGroupMemberURI_mixedMembers(t *testing.T) {
	client, asked := memberProbeServer(t, http.StatusNotFound, "sa-one", "sa-two")

	members := []string{"person-one", "sa-one", "person-two", "sa-two"}
	want := []string{
		client.ApiURL + "/users/person-one",
		client.ApiURL + "/service-accounts/sa-one",
		client.ApiURL + "/users/person-two",
		client.ApiURL + "/service-accounts/sa-two",
	}

	for i, member := range members {
		got, err := client.GroupMemberURI(member)
		if err != nil {
			t.Fatalf("GroupMemberURI(%q): unexpected error: %v", member, err)
		}
		if got != want[i] {
			t.Errorf("GroupMemberURI(%q) = %q, want %q", member, got, want[i])
		}
	}

	if len(asked()) != len(members) {
		t.Errorf("probed %d uids, want %d — every member's kind has to be asked for", len(asked()), len(members))
	}
}

// A failure that is not a 404 must abort the write. Falling back to /users/ would send a
// person's URI for a service account and turn an outage into a 400 naming the member.
func TestGroupMemberURI_nonNotFoundErrorAborts(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusUnauthorized, http.StatusForbidden} {
		client, _ := memberProbeServer(t, status)

		got, err := client.GroupMemberURI("some-uid")
		if err == nil {
			t.Errorf("status %d: GroupMemberURI returned %q and no error, want an error", status, got)
			continue
		}
		if !strings.Contains(err.Error(), "some-uid") {
			t.Errorf("status %d: error %q does not name the member", status, err)
		}
	}
}

// A Platform Manager old enough to have no /service-accounts route answers 404 to every
// probe, so every member is written exactly as the provider wrote it before service
// accounts existed.
func TestGroupMemberURI_platformManagerWithoutServiceAccounts(t *testing.T) {
	client, _ := memberProbeServer(t, http.StatusNotFound)

	got, err := client.GroupMemberURI("any-uid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := client.ApiURL + "/users/any-uid"; got != want {
		t.Errorf("GroupMemberURI = %q, want %q", got, want)
	}
}

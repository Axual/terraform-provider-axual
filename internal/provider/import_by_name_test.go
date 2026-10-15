package provider

import (
	webclient "axual-webclient"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeSearchAPI answers the searches import by name uses, ignoring case like Platform Manager: topic
// "payments" (t1), topic "twice" (two matches), the configs of t1 on environments "dev" (c1) and "prd"
// (c2), application "orders" (a1, a single object) and user "jane@example.com" (u1).
func fakeSearchAPI(t *testing.T) *webclient.Client {
	t.Helper()
	return fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/streams/search/findByName"):
			switch strings.ToLower(r.URL.Query().Get("name")) {
			case "payments":
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[{"uid":"t1","name":"payments"},{"uid":"t9","name":"Payments"}]}}`))
			case "twice":
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[{"uid":"a","name":"twice"},{"uid":"b","name":"twice"}]}}`))
			default:
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[]}}`))
			}
		case strings.HasSuffix(r.URL.Path, "/applications/search/findByName") && strings.EqualFold(r.URL.Query().Get("name"), "orders"):
			_, _ = w.Write([]byte(`{"uid":"a1","name":"orders"}`))
		case strings.HasSuffix(r.URL.Path, "/users/search/findByEmailAddress"):
			_, _ = w.Write([]byte(`{"_embedded":{"users":[{"uid":"u1","emailAddress":{"email":"jane@example.com"}}]}}`))
		case strings.HasSuffix(r.URL.Path, "/stream_configs/search/findByStream"):
			_, _ = w.Write([]byte(`{"_embedded":{"stream_configs":[
				{"uid":"c1","_embedded":{"environment":{"shortName":"dev"}}},
				{"uid":"c2","_embedded":{"environment":{"shortName":"prd"}}}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func importTopic(t *testing.T, id string) resource.ImportStateResponse {
	t.Helper()
	ctx := context.Background()
	r := &topicResource{provider: AxualProvider{client: fakeSearchAPI(t)}}
	resp := resource.ImportStateResponse{State: emptyState(t, r)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
	return resp
}

func importedID(t *testing.T, resp resource.ImportStateResponse) string {
	t.Helper()
	var id types.String
	resp.State.GetAttribute(context.Background(), path.Root("id"), &id)
	return id.ValueString()
}

// A topic is imported by name; the name must match exactly, case included.
func TestTopicImportByName(t *testing.T) {
	resp := importTopic(t, "name:payments")
	if resp.Diagnostics.HasError() {
		t.Fatalf("import error = %v", resp.Diagnostics)
	}
	if got := importedID(t, resp); got != "t1" {
		t.Errorf("id = %q, expected t1", got)
	}
}

func TestTopicImportByIDIsUnchanged(t *testing.T) {
	if got := importedID(t, importTopic(t, "abc123")); got != "abc123" {
		t.Errorf("id = %q, expected the ID as given", got)
	}
}

func TestImportByNameExplainsNoOrManyMatches(t *testing.T) {
	tests := map[string]string{
		"name:missing":  `no topic "missing" found`,
		"name:twice":    "2 topics match",
		"name:PAYMENTS": `no topic "PAYMENTS" found; it must match exactly, case included`,
	}
	for id, want := range tests {
		resp := importTopic(t, id)
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), want) {
			t.Errorf("import %s: diagnostics = %v, expected %q", id, resp.Diagnostics, want)
		}
	}
}

func TestFindTopicConfigIDPicksTheEnvironment(t *testing.T) {
	client := fakeSearchAPI(t)
	if id, err := findTopicConfigID(client, "payments/prd"); err != nil || id != "c2" {
		t.Errorf("payments/prd = (%q, %v), expected c2", id, err)
	}
	if _, err := findTopicConfigID(client, "payments/acc"); err == nil {
		t.Error("payments/acc found a config, expected none")
	}
	for _, bad := range []string{"payments", "/prd", "payments/"} {
		if _, err := findTopicConfigID(client, bad); err == nil || !strings.Contains(err.Error(), "<topic name>/<environment short name>") {
			t.Errorf("%q: error = %v, expected the expected format", bad, err)
		}
	}
}

// A UID never contains ":", so a prefix the resource does not take, or a typo, is not used as an ID.
func TestImportRejectsAnUnknownPrefix(t *testing.T) {
	resp := importTopic(t, "nme:payments")
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `expected an ID or name:<value>, got "nme:payments"`) {
		t.Errorf("diagnostics = %v, expected the unsupported import ID error", resp.Diagnostics)
	}
}

// Platform Manager answers the application search with one application, and 404 when there is none.
func TestFindApplicationIDByName(t *testing.T) {
	client := fakeSearchAPI(t)
	if id, err := findApplicationIDByName(client, "orders"); err != nil || id != "a1" {
		t.Errorf("orders = (%q, %v), expected a1", id, err)
	}
	if _, err := findApplicationIDByName(client, "missing"); err == nil || !strings.Contains(err.Error(), `no application "missing" found`) {
		t.Errorf("missing: error = %v, expected not found", err)
	}
}

func TestFindUserIDByEmail(t *testing.T) {
	client := fakeSearchAPI(t)
	if id, err := findUserIDByEmail(client, "jane@example.com"); err != nil || id != "u1" {
		t.Errorf("jane@example.com = (%q, %v), expected u1", id, err)
	}
	// An email address does not depend on case
	if id, err := findUserIDByEmail(client, "Jane@EXAMPLE.com"); err != nil || id != "u1" {
		t.Errorf("Jane@EXAMPLE.com = (%q, %v), expected u1", id, err)
	}
}

package provider

import (
	webclient "axual-webclient"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeSearchAPI answers the searches import by name uses: topic "payments" (t1), topic "twice"
// (two matches), and the configs of t1 on environments "dev" (c1) and "prd" (c2).
func fakeSearchAPI(t *testing.T) *webclient.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/streams/search/findByName"):
			switch r.URL.Query().Get("name") {
			case "payments":
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[{"uid":"t1","name":"payments"},{"uid":"t9","name":"Payments"}]}}`))
			case "twice":
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[{"uid":"a","name":"twice"},{"uid":"b","name":"twice"}]}}`))
			default:
				_, _ = w.Write([]byte(`{"_embedded":{"streams":[]}}`))
			}
		case strings.HasSuffix(r.URL.Path, "/stream_configs/search/findByStream"):
			_, _ = w.Write([]byte(`{"_embedded":{"stream_configs":[
				{"uid":"c1","_embedded":{"environment":{"shortName":"dev"}}},
				{"uid":"c2","_embedded":{"environment":{"shortName":"prd"}}}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return &webclient.Client{HTTPClient: server.Client(), ApiURL: server.URL}
}

func importTopic(t *testing.T, id string) resource.ImportStateResponse {
	t.Helper()
	ctx := context.Background()
	r := &topicResource{provider: AxualProvider{client: fakeSearchAPI(t)}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
	return resp
}

func importedID(t *testing.T, resp resource.ImportStateResponse) string {
	t.Helper()
	var id types.String
	resp.State.GetAttribute(context.Background(), path.Root("id"), &id)
	return id.ValueString()
}

// GitHub #114: a topic is imported by name; the name must match exactly, case included.
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
	tests := map[string]string{"name:missing": `no topic "missing" found`, "name:twice": "2 topics match"}
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

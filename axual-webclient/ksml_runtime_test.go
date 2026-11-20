package webclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const ksmlRuntimeAnswer = `{"id":"r1","name":"ksml","url":"https://ksml.example.com","ownerGroupId":"g1",` +
	`"authorizedGroups":[{"id":"g2","name":"B"}],"schemaRegistries":[{"type":"CONFLUENT","url":"https://sr.example.com"}]}`

func TestKsmlRuntimeCallsUseTheRuntimeEndpoints(t *testing.T) {
	request := KsmlRuntimeRequest{Name: "ksml", Url: "https://ksml.example.com", OwnerGroupId: "g1"}
	tests := []struct {
		name       string
		call       func(c *Client) (*KsmlRuntimeResponse, error)
		wantMethod string
		wantPath   string
		status     int
	}{
		{name: "create", wantMethod: "POST", wantPath: "/instances/i1/clusters/c1/ksml-runtimes", status: http.StatusCreated,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.CreateKsmlRuntime("i1", "c1", request) }},
		{name: "get", wantMethod: "GET", wantPath: "/instances/i1/clusters/c1/ksml-runtimes/r1", status: http.StatusOK,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.GetKsmlRuntime("i1", "c1", "r1") }},
		{name: "update", wantMethod: "PATCH", wantPath: "/instances/i1/clusters/c1/ksml-runtimes/r1", status: http.StatusOK,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.UpdateKsmlRuntime("i1", "c1", "r1", request) }},
		{name: "delete", wantMethod: "DELETE", wantPath: "/instances/i1/clusters/c1/ksml-runtimes/r1", status: http.StatusNoContent,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return nil, c.DeleteKsmlRuntime("i1", "c1", "r1") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method, path string
			var body []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path = r.Method, r.URL.Path
				body, _ = io.ReadAll(r.Body)
				w.WriteHeader(tt.status)
				if tt.status != http.StatusNoContent {
					_, _ = w.Write([]byte(ksmlRuntimeAnswer))
				}
			}))
			defer srv.Close()
			client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

			runtime, err := tt.call(client)

			if err != nil {
				t.Fatalf("error = %v, expected none", err)
			}
			if method != tt.wantMethod || path != tt.wantPath {
				t.Errorf("request = %s %s, expected %s %s", method, path, tt.wantMethod, tt.wantPath)
			}
			if tt.wantMethod == "DELETE" {
				return
			}
			if runtime.Uid != "r1" || runtime.OwnerGroupId != "g1" || len(runtime.AuthorizedGroups) != 1 ||
				runtime.AuthorizedGroups[0].Id != "g2" || len(runtime.SchemaRegistries) != 1 {
				t.Errorf("runtime = %+v, expected the answer mapped", runtime)
			}
			if tt.wantMethod == "GET" {
				return
			}
			var fields map[string]any
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatalf("request body %q: %v", body, err)
			}
			if fields["ownerGroupId"] != "g1" {
				t.Errorf("ownerGroupId = %v, expected g1 (body %s)", fields["ownerGroupId"], body)
			}
		})
	}
}

// The PATCH only applies the fields in the body, so a removed description or schema registry list
// is an explicit null, and the authorized groups are left out unless the configuration has them.
func TestKsmlRuntimeRequestBody(t *testing.T) {
	fields := func(t *testing.T, request KsmlRuntimeRequest) map[string]any {
		t.Helper()
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}

	t.Run("unset description and schema registries are null", func(t *testing.T) {
		body := fields(t, KsmlRuntimeRequest{Name: "ksml"})
		for _, key := range []string{"description", "schemaRegistries"} {
			if value, present := body[key]; !present || value != nil {
				t.Errorf("%s = %v (present %t), expected an explicit null", key, value, present)
			}
		}
	})
	t.Run("nil authorized groups are left out", func(t *testing.T) {
		if _, present := fields(t, KsmlRuntimeRequest{})["authorizedGroupIds"]; present {
			t.Error("authorizedGroupIds was sent, expected it left out")
		}
	})
	t.Run("empty authorized groups are an empty list", func(t *testing.T) {
		ids, present := fields(t, KsmlRuntimeRequest{AuthorizedGroupIds: &[]string{}})["authorizedGroupIds"]
		if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
			t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list", ids, present)
		}
	})
}

package webclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const ksmlRuntimeAnswer = `{"id":"r1","type":"KSML","name":"ksml","instanceId":"i1","clusterId":"c1","ownerGroupId":"g1",` +
	`"authorizedGroups":[{"id":"g2","name":"B"}],` +
	`"config":{"url":"https://ksml.example.com","schemaRegistries":[{"type":"CONFLUENT","url":"https://sr.example.com"}]}}`

func jsonFields(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("request body %q: %v", body, err)
	}
	return fields
}

func TestKsmlRuntimeCallsUseTheRuntimeEndpoints(t *testing.T) {
	config := KsmlRuntimeConfig{Url: "https://ksml.example.com", SchemaRegistries: []SchemaRegistryConfig{}}
	create := KsmlRuntimeCreateRequest{Name: "ksml", InstanceId: "i1", ClusterId: "c1", OwnerGroupId: "g1", Config: config}
	update := KsmlRuntimeUpdateRequest{OwnerGroupId: "g1", Config: config}
	tests := []struct {
		name       string
		call       func(c *Client) (*KsmlRuntimeResponse, error)
		wantMethod string
		wantPath   string
		status     int
	}{
		{name: "create", wantMethod: "POST", wantPath: "/runtimes", status: http.StatusCreated,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.CreateKsmlRuntime(create) }},
		{name: "get", wantMethod: "GET", wantPath: "/runtimes/r1", status: http.StatusOK,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.GetKsmlRuntime("r1") }},
		{name: "update", wantMethod: "PATCH", wantPath: "/runtimes/r1", status: http.StatusOK,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return c.UpdateKsmlRuntime("r1", update) }},
		{name: "delete", wantMethod: "DELETE", wantPath: "/runtimes/r1", status: http.StatusNoContent,
			call: func(c *Client) (*KsmlRuntimeResponse, error) { return nil, c.DeleteKsmlRuntime("r1") }},
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
				runtime.AuthorizedGroups[0].Id != "g2" || runtime.Config.Url != "https://ksml.example.com" ||
				len(runtime.Config.SchemaRegistries) != 1 {
				t.Errorf("runtime = %+v, expected the answer mapped", runtime)
			}
			if tt.wantMethod == "GET" {
				return
			}
			fields := jsonFields(t, body)
			wantConfig := map[string]any{"url": "https://ksml.example.com", "schemaRegistries": []any{}}
			if fields["ownerGroupId"] != "g1" || !reflect.DeepEqual(fields["config"], wantConfig) {
				t.Errorf("ownerGroupId = %v, config = %v; expected g1 and %v (body %s)", fields["ownerGroupId"], fields["config"], wantConfig, body)
			}
		})
	}
}

// A create names the type, the instance and the cluster in the body, and leaves out what has no value.
func TestKsmlRuntimeCreateRequestBody(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(ksmlRuntimeAnswer))
	}))
	defer srv.Close()
	client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}

	if _, err := client.CreateKsmlRuntime(KsmlRuntimeCreateRequest{Name: "ksml", InstanceId: "i1", ClusterId: "c1", OwnerGroupId: "g1"}); err != nil {
		t.Fatalf("error = %v, expected none", err)
	}

	fields := jsonFields(t, body)
	for key, want := range map[string]string{"type": "KSML", "name": "ksml", "instanceId": "i1", "clusterId": "c1"} {
		if fields[key] != want {
			t.Errorf("%s = %v, expected %s (body %s)", key, fields[key], want, body)
		}
	}
	for _, key := range []string{"description", "authorizedGroupIds"} {
		if _, present := fields[key]; present {
			t.Errorf("%s was sent, expected it left out (body %s)", key, body)
		}
	}
}

// The PATCH only applies the fields in the body, so a removed description is an explicit null, the
// authorized groups are left out unless the configuration has them, and the fixed fields are not sent.
func TestKsmlRuntimeUpdateRequestBody(t *testing.T) {
	fields := func(t *testing.T, request KsmlRuntimeUpdateRequest) map[string]any {
		t.Helper()
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		return jsonFields(t, body)
	}

	t.Run("an unset description is null", func(t *testing.T) {
		if value, present := fields(t, KsmlRuntimeUpdateRequest{})["description"]; !present || value != nil {
			t.Errorf("description = %v (present %t), expected an explicit null", value, present)
		}
	})
	t.Run("the fixed fields are not sent", func(t *testing.T) {
		body := fields(t, KsmlRuntimeUpdateRequest{OwnerGroupId: "g1"})
		for _, key := range []string{"name", "type", "instanceId", "clusterId"} {
			if _, present := body[key]; present {
				t.Errorf("%s was sent, expected it left out", key)
			}
		}
	})
	t.Run("nil authorized groups are left out", func(t *testing.T) {
		if _, present := fields(t, KsmlRuntimeUpdateRequest{})["authorizedGroupIds"]; present {
			t.Error("authorizedGroupIds was sent, expected it left out")
		}
	})
	t.Run("empty authorized groups are an empty list", func(t *testing.T) {
		ids, present := fields(t, KsmlRuntimeUpdateRequest{AuthorizedGroupIds: &[]string{}})["authorizedGroupIds"]
		if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
			t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list", ids, present)
		}
	})
}

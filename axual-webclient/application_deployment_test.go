package webclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUidFromLocationHeader(t *testing.T) {
	testCases := []struct {
		desc     string
		location string
		expected string
	}{
		{
			desc:     "Location header of a Flink SQL application deployment",
			location: "https://platform.local/api/application_deployments/e348ea9cd4a84f86b4db847aa3a91d68",
			expected: "e348ea9cd4a84f86b4db847aa3a91d68",
		},
		{
			desc:     "Location header of a KSML application deployment",
			location: "https://platform.local/api/application_deployments/c3741bc7ea3e418bbad2e142bb05b98b",
			expected: "c3741bc7ea3e418bbad2e142bb05b98b",
		},
		{
			desc:     "Location header of a Connector application deployment",
			location: "https://platform.local/api/application_deployments/c94deccbcf7f4841a426196eaa19f487",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "trailing slash is ignored",
			location: "https://platform.local/api/application_deployments/c94deccbcf7f4841a426196eaa19f487/",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "relative Location header",
			location: "/api/application_deployments/c94deccbcf7f4841a426196eaa19f487",
			expected: "c94deccbcf7f4841a426196eaa19f487",
		},
		{
			desc:     "no Location header",
			location: "",
			expected: "",
		},
	}
	for _, c := range testCases {
		t.Run(c.desc, func(t *testing.T) {
			headers := http.Header{}
			if c.location != "" {
				headers.Set("Location", c.location)
			}
			if uid := uidFromLocationHeader(headers); uid != c.expected {
				t.Fatalf("expected Location %q to yield Uid %q, got %q", c.location, c.expected, uid)
			}
		})
	}
}

func TestUidFromLocationHeaderWithoutHeaders(t *testing.T) {
	if uid := uidFromLocationHeader(nil); uid != "" {
		t.Fatalf("expected no Uid for nil headers, got %q", uid)
	}
}

func TestApplicationDeploymentUpdateRequestClearTarget(t *testing.T) {
	body, err := json.Marshal(ApplicationDeploymentUpdateRequest{Configs: map[string]string{"a": "b"}, ClearTarget: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); got != `{"configs":{"a":"b"},"targetId":null,"targetVersion":null}` {
		t.Errorf("body = %s", got)
	}
	body, _ = json.Marshal(ApplicationDeploymentUpdateRequest{Configs: map[string]string{}, TargetId: "cc1"})
	if got := string(body); got != `{"configs":{},"targetId":"cc1"}` {
		t.Errorf("body = %s", got)
	}
}

// TestUpdateApplicationDeploymentFallsBackToPut covers Platform Manager 15.0.x, which has no PATCH
// endpoint for a deployment and refuses the body with 400 "Could not read payload".
func TestUpdateApplicationDeploymentFallsBackToPut(t *testing.T) {
	const pm15Answer = `{"detail":"Invalid request body: ` + "`Could not read payload`" + `. Please check API docs for building correct request."}`
	tests := []struct {
		name        string
		patchStatus int
		patchBody   string
		request     ApplicationDeploymentUpdateRequest
		wantMethods []string
		wantErr     bool
	}{
		{name: "PM 16: PATCH works", patchStatus: http.StatusNoContent,
			request: ApplicationDeploymentUpdateRequest{Configs: map[string]string{"a": "b"}}, wantMethods: []string{"PATCH"}},
		{name: "PM 15: configs only falls back to PUT", patchStatus: http.StatusBadRequest, patchBody: pm15Answer,
			request: ApplicationDeploymentUpdateRequest{Configs: map[string]string{"a": "b"}}, wantMethods: []string{"PATCH", "PUT"}},
		{name: "PM 15: a target change is not sent with PUT", patchStatus: http.StatusBadRequest, patchBody: pm15Answer,
			request: ApplicationDeploymentUpdateRequest{Configs: map[string]string{"a": "b"}, TargetId: "kc"}, wantMethods: []string{"PATCH"}, wantErr: true},
		{name: "another 400 is returned as is", patchStatus: http.StatusBadRequest, patchBody: `{"detail":"invalid config"}`,
			request: ApplicationDeploymentUpdateRequest{Configs: map[string]string{"a": "b"}}, wantMethods: []string{"PATCH"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var methods []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.Method == "PATCH" {
					w.WriteHeader(tt.patchStatus)
					_, _ = w.Write([]byte(tt.patchBody))
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()
			client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}
			_, err := client.UpdateApplicationDeployment("dep1", tt.request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if strings.Join(methods, ",") != strings.Join(tt.wantMethods, ",") {
				t.Fatalf("methods = %v, want %v", methods, tt.wantMethods)
			}
		})
	}
}

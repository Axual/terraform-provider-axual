package webclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAll(t *testing.T) {
	tests := []struct {
		desc      string
		pages     []string
		wantUIDs  []string
		wantCalls int
	}{
		{
			desc: "paged HAL response, three pages",
			pages: []string{
				`{"_embedded":{"streams":[{"uid":"a"},{"uid":"b"}]},"page":{"totalPages":3}}`,
				`{"_embedded":{"streams":[{"uid":"c"}]},"page":{"totalPages":3}}`,
				`{"_embedded":{"streams":[{"uid":"d"}]},"page":{"totalPages":3}}`,
			},
			wantUIDs:  []string{"a", "b", "c", "d"},
			wantCalls: 3,
		},
		{
			desc:      "empty paged HAL response",
			pages:     []string{`{"_embedded":{"streams":[]},"page":{"totalPages":0}}`},
			wantUIDs:  nil,
			wantCalls: 1,
		},
		{
			desc:      "unpaged HAL response",
			pages:     []string{`{"_embedded":{"stream_configs":[{"uid":"x"},{"uid":"y"}]}}`},
			wantUIDs:  []string{"x", "y"},
			wantCalls: 1,
		},
		{
			desc:      "plain JSON array",
			pages:     []string{`[{"uid":"p"},{"uid":"q"}]`},
			wantUIDs:  []string{"p", "q"},
			wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("page"); got != fmt.Sprint(calls) {
					t.Errorf("page = %q, want %d", got, calls)
				}
				if got := r.URL.Query().Get("name"); got != "orders" {
					t.Errorf("name = %q, want the filter to be passed on", got)
				}
				_, _ = w.Write([]byte(tt.pages[calls]))
				calls++
			}))
			defer srv.Close()

			client := &Client{HTTPClient: srv.Client(), ApiURL: srv.URL}
			items, err := client.ListAll("streams/search/findByAttributes", map[string][]string{"name": {"orders"}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var uids []string
			for _, it := range items {
				uids = append(uids, it.String("uid"))
			}
			if strings.Join(uids, ",") != strings.Join(tt.wantUIDs, ",") {
				t.Errorf("uids = %v, want %v", uids, tt.wantUIDs)
			}
			if calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestListItemEmbedded(t *testing.T) {
	hal := ListItem{"_embedded": map[string]any{"environment": map[string]any{"uid": "env1"}}}
	plain := ListItem{"environment": map[string]any{"uid": "env2"}}
	missing := ListItem{"uid": "x"}

	if got := hal.Embedded("environment", "uid"); got != "env1" {
		t.Errorf("HAL item: got %q, want env1", got)
	}
	if got := plain.Embedded("environment", "uid"); got != "env2" {
		t.Errorf("plain item: got %q, want env2", got)
	}
	if got := missing.Embedded("environment", "uid"); got != "" {
		t.Errorf("missing object: got %q, want empty", got)
	}
}

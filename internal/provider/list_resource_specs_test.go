package provider

import (
	webclient "axual-webclient"
	"strings"
	"testing"
)

func TestResolveOne(t *testing.T) {
	items := []webclient.ListItem{
		{"uid": "a", "name": "orders"},
		{"uid": "b", "name": "orders-archive"},
	}
	exact := func(it webclient.ListItem) bool { return it.String("name") == "orders" }

	if id, err := resolveOne("topic", "orders", items, exact); err != nil || id != "a" {
		t.Errorf("one match: got %q, %v; want a", id, err)
	}
	if _, err := resolveOne("topic", "orders", items, nil); err == nil || !strings.Contains(err.Error(), "more than one topic") {
		t.Errorf("two matches: got %v, want an error about more than one", err)
	}
	if _, err := resolveOne("topic", "payments", nil, nil); err == nil || !strings.Contains(err.Error(), `no topic found with ID or name "payments"`) {
		t.Errorf("no match: got %v, want a not-found error", err)
	}
}

func TestSameJSON(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{`{"type":"record","name":"A"}`, "{\n  \"name\": \"A\",\n  \"type\": \"record\"\n}", true},
		{`{"name":"A"}`, `{"name":"B"}`, false},
		{`syntax = "proto3";`, `syntax = "proto3";`, false}, // not JSON
	}
	for _, tt := range tests {
		if got := sameJSON(tt.a, tt.b); got != tt.want {
			t.Errorf("sameJSON(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

package provider

import (
	webclient "axual-webclient"
	"testing"
)

func TestDecisionFieldsOf(t *testing.T) {
	decided := webclient.GrantView{Comment: "Not allowed, INC-1234", ProcessedAt: "2026-10-26T09:45:43", ProcessedBy: "tom@axual.com"}
	tests := []struct {
		status                 string
		wantReason, wantAt, by bool
	}{
		{"Rejected", true, true, true},
		{"Revoked", true, true, true},
		{"Approved", false, true, true},
		// A reopened grant keeps the old comment and processed_at; they must not show.
		{"Pending", false, false, false},
		{"Cancelled", false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			g := decided
			g.Status = tt.status
			f := decisionFieldsOf(g)
			if f.DecisionReason.IsNull() == tt.wantReason || f.DecidedAt.IsNull() == tt.wantAt || f.DecidedBy.IsNull() == tt.by {
				t.Errorf("got %+v", f)
			}
		})
	}
}

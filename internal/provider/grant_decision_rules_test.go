package provider

import (
	"testing"

	webclient "axual-webclient"
)

func TestDecide(t *testing.T) {
	// Design v8 section 4.2: the whole rule. Reopened only adds a warning.
	tests := []struct {
		desc string
		in   decisionInput
		want decisionOutcome
	}{
		{"APPROVED before the request waits for it",
			decisionInput{Status: statusNotFound, Decision: decisionApproved},
			decisionOutcome{PlannedStatus: statusNotFound, Warning: warnWaitingForRequest}},
		{"APPROVED approves a first request",
			decisionInput{Status: "Pending", Decision: decisionApproved},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved"}},
		{"APPROVED approves a request made again after any close, with a warning",
			decisionInput{Status: "Pending", Decision: decisionApproved, Reopened: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", Warning: warnApprovedAgain}},
		{"APPROVED: Approved needs nothing",
			decisionInput{Status: "Approved", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Approved"}},
		{"APPROVED leaves a Rejected grant alone, with a warning",
			decisionInput{Status: "Rejected", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Rejected", Warning: warnClosed}},
		{"APPROVED leaves a Revoked grant alone, with a warning",
			decisionInput{Status: "Revoked", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Revoked", Warning: warnClosed}},
		{"APPROVED leaves a Cancelled grant alone, with a warning",
			decisionInput{Status: "Cancelled", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Cancelled", Warning: warnClosed}},
		{"DENIED before the request does nothing",
			decisionInput{Status: statusNotFound, Decision: decisionDenied},
			decisionOutcome{PlannedStatus: statusNotFound}},
		{"DENIED rejects a Pending request",
			decisionInput{Status: "Pending", Decision: decisionDenied},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}},
		{"DENIED rejects a request made again, with a warning",
			decisionInput{Status: "Pending", Decision: decisionDenied, Reopened: true},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected", Warning: warnDeniedAgain}},
		{"DENIED revokes an Approved grant",
			decisionInput{Status: "Approved", Decision: decisionDenied},
			decisionOutcome{Action: actionRevoke, PlannedStatus: "Revoked"}},
		{"DENIED leaves a Rejected grant alone",
			decisionInput{Status: "Rejected", Decision: decisionDenied},
			decisionOutcome{PlannedStatus: "Rejected"}},
		{"DENIED leaves a Revoked grant alone",
			decisionInput{Status: "Revoked", Decision: decisionDenied},
			decisionOutcome{PlannedStatus: "Revoked"}},
		{"DENIED leaves a Cancelled grant alone",
			decisionInput{Status: "Cancelled", Decision: decisionDenied},
			decisionOutcome{PlannedStatus: "Cancelled"}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := decide(tt.in); got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestCloseKey(t *testing.T) {
	first := webclient.GrantView{Status: "Revoked", ProcessedAt: "2026-10-27T13:07:20", ProcessedBy: "Tom@axual.com"}
	reopenedGrant := first
	reopenedGrant.Status = "Pending" // a request made again keeps who closed it, and when
	later := first
	later.ProcessedAt = "2026-10-28T09:00:00"

	if closeKey(webclient.GrantView{Status: "Pending"}) != "" {
		t.Error("a grant that was never decided has no close")
	}
	if closeKey(first) != closeKey(reopenedGrant) {
		t.Error("a request made again must keep the close it was made after")
	}
	if closeKey(first) == closeKey(later) {
		t.Error("a later close must differ from an earlier one")
	}
	if !reopened(reopenedGrant) || reopened(webclient.GrantView{Status: "Pending"}) || reopened(first) {
		t.Error("reopened: only a Pending grant that was closed before")
	}
}

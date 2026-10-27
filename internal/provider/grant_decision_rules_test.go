package provider

import (
	"testing"

	webclient "axual-webclient"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		desc string
		in   decisionInput
		want decisionOutcome
	}{
		{"first request is approved",
			decisionInput{Status: "Pending", Decision: decisionApproved},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved"}},
		{"decision before the request waits for it",
			decisionInput{Status: statusNotFound, Decision: decisionApproved},
			decisionOutcome{PlannedStatus: statusNotFound, Warning: warnWaitingForRequest}},
		{"request made again after a close by someone else waits",
			decisionInput{Status: "Pending", Decision: decisionApproved, Reopened: true},
			decisionOutcome{PlannedStatus: "Pending", Warning: warnReopened}},
		{"approve_again or decision change accepts the close and approves",
			decisionInput{Status: "Pending", Decision: decisionApproved, Reopened: true, Accepted: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved"}},
		{"request made again after the application team's own close is approved again",
			decisionInput{Status: "Pending", Decision: decisionApproved, Reopened: true, OwnClose: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", Warning: warnApprovedAgain}},
		{"Approved: nothing to do",
			decisionInput{Status: "Approved", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Approved"}},
		{"closed by someone else: the next request will wait",
			decisionInput{Status: "Revoked", Decision: decisionApproved},
			decisionOutcome{PlannedStatus: "Revoked", Warning: warnClosed}},
		{"closed and accepted: the next request will be approved",
			decisionInput{Status: "Rejected", Decision: decisionApproved, Accepted: true},
			decisionOutcome{PlannedStatus: "Rejected", Warning: warnClosed, NextApproved: true}},
		{"closed by the application team itself: the next request will be approved",
			decisionInput{Status: "Cancelled", Decision: decisionApproved, OwnClose: true},
			decisionOutcome{PlannedStatus: "Cancelled", Warning: warnClosed, NextApproved: true}},
		{"DENIED rejects a Pending request",
			decisionInput{Status: "Pending", Decision: decisionDenied},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}},
		{"DENIED rejects a request made again, with a warning",
			decisionInput{Status: "Pending", Decision: decisionDenied, Reopened: true, OwnClose: true},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected", Warning: warnDeniedAgain}},
		{"DENIED revokes an Approved grant",
			decisionInput{Status: "Approved", Decision: decisionDenied},
			decisionOutcome{Action: actionRevoke, PlannedStatus: "Revoked"}},
		{"DENIED leaves a closed grant alone",
			decisionInput{Status: "Cancelled", Decision: decisionDenied},
			decisionOutcome{PlannedStatus: "Cancelled"}},
		{"DENIED before the request does nothing",
			decisionInput{Status: statusNotFound, Decision: decisionDenied},
			decisionOutcome{PlannedStatus: statusNotFound}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := decide(tt.in); got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestOwnClose(t *testing.T) {
	tests := []struct {
		desc                         string
		processedBy, requestedBy, me string
		want                         bool
	}{
		{"same user closed and asked again", "alex@axual.com", "Alex@axual.com", "tom@axual.com", true},
		{"topic owner closed", "tom@axual.com", "alex@axual.com", "tom@axual.com", false},
		{"another application-team user asked again", "alex@axual.com", "ci@axual.com", "tom@axual.com", false},
		{"one shared account", "ci@axual.com", "ci@axual.com", "ci@axual.com", false},
		{"signed-in user unknown", "alex@axual.com", "alex@axual.com", "", false},
		{"never closed", "", "alex@axual.com", "tom@axual.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := ownClose(tt.processedBy, tt.requestedBy, tt.me); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
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
		t.Error("a later close must differ from an accepted earlier one")
	}
	if !reopened(reopenedGrant) || reopened(webclient.GrantView{Status: "Pending"}) || reopened(first) {
		t.Error("reopened: only a Pending grant that was closed before")
	}
}

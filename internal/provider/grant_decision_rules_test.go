package provider

import "testing"

func TestDecide(t *testing.T) {
	tests := []struct {
		desc string
		in   decisionInput
		want decisionOutcome
	}{
		{"new block approves the Pending request",
			decisionInput{Status: "Pending", Decision: decisionApproved, Create: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview}},
		{"reopened request waits",
			decisionInput{Status: "Pending", Decision: decisionApproved, NextRequest: nextRequestReview},
			decisionOutcome{PlannedStatus: "Pending", NextRequest: nextRequestReview, Warning: warnReopened}},
		{"review change approves the reopened request",
			decisionInput{Status: "Pending", Decision: decisionApproved, ReviewChanged: true, NextRequest: nextRequestReview},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview}},
		{"pre-approved request is approved and uses up the pre-approval",
			decisionInput{Status: "Pending", Decision: decisionApproved, NextRequest: nextRequestApprove},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview}},
		{"topic PR first: waiting, next request pre-approved",
			decisionInput{Status: statusNotFound, Decision: decisionApproved, Create: true},
			decisionOutcome{PlannedStatus: statusNotFound, NextRequest: nextRequestApprove, Warning: warnWaitingForRequest}},
		{"review change while closed doesn't pre-approve",
			decisionInput{Status: "Revoked", Decision: decisionApproved, ReviewChanged: true, NextRequest: nextRequestReview},
			decisionOutcome{PlannedStatus: "Revoked", NextRequest: nextRequestReview, Warning: warnClosed}},
		{"decision DENIED to APPROVED while closed pre-approves",
			decisionInput{Status: "Rejected", Decision: decisionApproved, DecisionChanged: true},
			decisionOutcome{PlannedStatus: "Rejected", NextRequest: nextRequestApprove, Warning: warnClosed}},
		{"approved elsewhere uses up the pre-approval",
			decisionInput{Status: "Approved", Decision: decisionApproved, NextRequest: nextRequestApprove},
			decisionOutcome{PlannedStatus: "Approved", NextRequest: nextRequestReview}},
		{"import reviews a Pending request",
			decisionInput{Status: "Pending", Decision: decisionApproved, DecisionChanged: true, Imported: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview}},
		{"import never pre-approves a revoked grant",
			decisionInput{Status: "Revoked", Decision: decisionApproved, DecisionChanged: true, Imported: true},
			decisionOutcome{PlannedStatus: "Revoked", NextRequest: nextRequestReview, Warning: warnClosed}},
		{"request made again after the application team's own close is approved again",
			decisionInput{Status: "Pending", Decision: decisionApproved, NextRequest: nextRequestReview, ClosedByRequester: true},
			decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview, Warning: warnApprovedAgain}},
		{"DENIED still rejects after the application team's own close",
			decisionInput{Status: "Pending", Decision: decisionDenied, ClosedByRequester: true},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}},
		{"DENIED rejects a Pending request without review",
			decisionInput{Status: "Pending", Decision: decisionDenied},
			decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}},
		{"DENIED revokes an Approved grant",
			decisionInput{Status: "Approved", Decision: decisionDenied},
			decisionOutcome{Action: actionRevoke, PlannedStatus: "Revoked"}},
		{"DENIED leaves a closed grant alone",
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

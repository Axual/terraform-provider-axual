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
	sameSecond := first
	sameSecond.Comment = "rejected in the UI"
	if closeKey(first) == closeKey(sameSecond) {
		t.Error("two closes by one user within a second must differ")
	}
	if !reopened(reopenedGrant) || reopened(webclient.GrantView{Status: "Pending"}) || reopened(first) {
		t.Error("reopened: only a Pending grant that was closed before")
	}
}

func TestClosedOutside(t *testing.T) {
	tests := []struct {
		desc                 string
		decision, status     string
		closeKey, ownClose   string
		byRequester, outside bool
	}{
		{"UI revoke under APPROVED", decisionApproved, "Revoked", "t2|tom", "", false, true},
		{"UI reject after an own reject", decisionApproved, "Rejected", "t3|tom", "t1|tom", false, true},
		{"own reject, then the flip to APPROVED", decisionApproved, "Rejected", "t1|tom", "t1|tom", false, false},
		{"the application team gave up access", decisionApproved, "Revoked", "t2|alex", "", true, false},
		{"no stored key (import, moved block)", decisionApproved, "Rejected", "t1|tom", "", false, true},
		{"Cancelled", decisionApproved, "Cancelled", "t2|admin", "", false, false},
		{"Approved", decisionApproved, "Approved", "t2|tom", "", false, false},
		{"Pending", decisionApproved, "Pending", "t2|tom", "", false, false},
		{"under DENIED", decisionDenied, "Revoked", "t2|tom", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := closedOutside(tt.decision, tt.status, tt.closeKey, tt.ownClose, tt.byRequester); got != tt.outside {
				t.Errorf("got %v, want %v", got, tt.outside)
			}
		})
	}
	if !closedByRequester(webclient.GrantView{ProcessedBy: "Alex@axual.com", RequestedBy: "alex@axual.com"}) ||
		closedByRequester(webclient.GrantView{ProcessedBy: "tom@axual.com", RequestedBy: "alex@axual.com"}) ||
		closedByRequester(webclient.GrantView{}) {
		t.Error("closedByRequester")
	}
}

func TestAdoptClose(t *testing.T) {
	rejected := &webclient.GrantView{Status: "Rejected", ProcessedAt: "t1", ProcessedBy: "tom", Comment: "no"}
	tests := []struct {
		desc, decision string
		g              *webclient.GrantView
		own, want      string
	}{
		{"DENIED, moved or imported: the close becomes our own", decisionDenied, rejected, "", "t1|tom|no"},
		{"DENIED, a stale key is replaced", decisionDenied, rejected, "t0|tom|old", "t1|tom|no"},
		{"APPROVED keeps the stored key: a UI revoke stays outside", decisionApproved, rejected, "t0|tom|old", "t0|tom|old"},
		{"DENIED on a Pending grant keeps the key", decisionDenied, &webclient.GrantView{Status: "Pending", ProcessedBy: "tom"}, "k", "k"},
		{"no grant", decisionDenied, nil, "k", "k"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := adoptClose(tt.decision, tt.g, tt.own); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
	// The reviewer's case: a moved rejection, then the flip to APPROVED, isn't reported.
	own := adoptClose(decisionDenied, rejected, "")
	if closedOutside(decisionApproved, "Rejected", closeKey(*rejected), own, false) {
		t.Error("the flip after a moved rejection is reported as closed outside Terraform")
	}
}

func TestOtherApplication(t *testing.T) {
	tests := []struct {
		desc                     string
		pinned, seen, configured string
		other                    bool
	}{
		{"the same application", "a1", "a1", "Log Scraper", false},
		{"re-created with the same name", "a1", "b2", "Log Scraper", true},
		{"confirmed by setting the new UID", "a1", "b2", "b2", false},
		{"confirmed, other case", "a1", "B2", "b2", false},
		{"nothing pinned yet (topic first)", "", "b2", "Log Scraper", false},
		{"no grant found", "a1", "", "Log Scraper", false},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := otherApplication(tt.pinned, tt.seen, tt.configured); got != tt.other {
				t.Errorf("got %v, want %v", got, tt.other)
			}
		})
	}
}

func TestWithOffer(t *testing.T) {
	approve := decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", Warning: warnApprovedAgain}
	if got := withOffer(approve, "Pending", true); got != approve {
		t.Errorf("offered: got %+v", got)
	}
	if got, want := withOffer(approve, "Pending", false), (decisionOutcome{PlannedStatus: "Pending", Warning: warnApproveNotOffered}); got != want {
		t.Errorf("not offered: got %+v, want %+v", got, want)
	}
	reject := decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}
	if got := withOffer(reject, "Pending", false); got != reject {
		t.Errorf("only approve is strict: got %+v", got)
	}
}

func TestRemovedByTerraform(t *testing.T) {
	for _, tc := range []struct {
		status, comment string
		want            bool
	}{
		{"Revoked", "Decision removed from Terraform", true},
		{"Revoked", "Decision removed from Terraform: INC-1", true},
		{"Revoked", "Revoked in terraform", false},
		{"Rejected", "Decision removed from Terraform", false},
		{"Revoked", "", false},
	} {
		if got := removedByTerraform(webclient.GrantView{Status: tc.status, Comment: tc.comment}); got != tc.want {
			t.Errorf("%s %q: got %v, want %v", tc.status, tc.comment, got, tc.want)
		}
	}
}

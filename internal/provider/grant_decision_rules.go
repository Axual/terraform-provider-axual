package provider

import (
	"strings"

	webclient "axual-webclient"
)

// The rules of axual_application_access_grant_decision, without Terraform or API types, so they
// can be tested on their own.

const (
	decisionApproved = "APPROVED"
	decisionDenied   = "DENIED"

	statusNotFound = "Not found"

	actionNone    = ""
	actionApprove = "approve"
	actionReject  = "reject"
	actionRevoke  = "revoke"
)

// decisionInput is what the rules look at for one decision in one plan or apply.
type decisionInput struct {
	Status   string // the grant's status, or statusNotFound
	Decision string // decisionApproved or decisionDenied
	// Reopened is true for a Pending grant that was closed before (rejected, revoked or cancelled)
	// and requested again.
	Reopened bool
	// OwnClose is true when the application team closed the grant itself: see ownClose.
	OwnClose bool
	// Accepted is true when the topic team accepted the last close: decision or approve_again
	// changed in this plan, or changed while the grant had this close. A create or an import
	// doesn't count.
	Accepted bool
}

// decisionOutcome is what the provider does and plans.
type decisionOutcome struct {
	Action        string
	PlannedStatus string
	Warning       string // one of the warn* kinds, or ""
	// NextApproved tells, for a closed grant, whether a new request will be approved.
	NextApproved bool
}

const (
	warnWaitingForRequest = "waiting"
	warnReopened          = "reopened"
	warnClosed            = "closed"
	warnApprovedAgain     = "approved again"
	warnDeniedAgain       = "denied again"
)

// decide applies the standing decision:
//
//   - DENIED rejects a Pending request and revokes an Approved grant.
//   - APPROVED approves every Pending request, except one made again after someone other than the
//     application team closed the grant: that one waits until the topic team accepts the close by
//     changing decision or approve_again.
func decide(in decisionInput) decisionOutcome {
	if in.Decision == decisionDenied {
		switch in.Status {
		case "Pending":
			out := decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}
			if in.Reopened {
				out.Warning = warnDeniedAgain
			}
			return out
		case "Approved":
			return decisionOutcome{Action: actionRevoke, PlannedStatus: "Revoked"}
		}
		return decisionOutcome{PlannedStatus: in.Status}
	}

	switch in.Status {
	case "Pending":
		switch {
		case !in.Reopened || in.Accepted:
			return decisionOutcome{Action: actionApprove, PlannedStatus: "Approved"}
		case in.OwnClose:
			return decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", Warning: warnApprovedAgain}
		}
		return decisionOutcome{PlannedStatus: "Pending", Warning: warnReopened}
	case "Approved":
		return decisionOutcome{PlannedStatus: "Approved"}
	case statusNotFound:
		return decisionOutcome{PlannedStatus: in.Status, Warning: warnWaitingForRequest}
	default: // Rejected, Revoked, Cancelled
		return decisionOutcome{PlannedStatus: in.Status, Warning: warnClosed, NextApproved: in.Accepted || in.OwnClose}
	}
}

// ownClose reports whether the application team closed the grant itself: the user who closed it
// last (processedBy) is the user who requested it (requestedBy), and that user isn't the signed-in
// user (the topic side's own account). With one account for both sides, or when anything is unknown,
// it is false, so a request made again waits for the topic team.
func ownClose(processedBy, requestedBy, me string) bool {
	return processedBy != "" && me != "" &&
		strings.EqualFold(processedBy, requestedBy) && !strings.EqualFold(processedBy, me)
}

// reopened reports whether a Pending grant was closed before: the platform keeps who closed it
// last when the grant is requested again.
func reopened(g webclient.GrantView) bool {
	return g.Status == "Pending" && g.ProcessedBy != ""
}

// closeKey identifies the grant's last decision or close (who and when), so the provider can tell
// whether the topic team accepted this close or an earlier one. "" when the grant was never decided.
func closeKey(g webclient.GrantView) string {
	if g.ProcessedBy == "" {
		return ""
	}
	return g.ProcessedAt + "|" + strings.ToLower(g.ProcessedBy)
}

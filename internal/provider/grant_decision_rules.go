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
	// and requested again. It only changes the warning.
	Reopened bool
}

// decisionOutcome is what the provider does and plans.
type decisionOutcome struct {
	Action        string
	PlannedStatus string
	Warning       string // one of the warn* kinds, or ""
}

const (
	warnWaitingForRequest = "waiting"
	warnClosed            = "closed"
	warnApprovedAgain     = "approved again"
	warnDeniedAgain       = "denied again"
)

// decide applies the standing decision. It looks only at the grant's current status, never at who
// closed it, so nothing the application team does can change the outcome:
//
//   - DENIED rejects a Pending request and revokes an Approved grant.
//   - APPROVED approves every Pending request.
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
		out := decisionOutcome{Action: actionApprove, PlannedStatus: "Approved"}
		if in.Reopened {
			out.Warning = warnApprovedAgain
		}
		return out
	case "Approved":
		return decisionOutcome{PlannedStatus: "Approved"}
	case statusNotFound:
		return decisionOutcome{PlannedStatus: in.Status, Warning: warnWaitingForRequest}
	default: // Rejected, Revoked, Cancelled
		return decisionOutcome{PlannedStatus: in.Status, Warning: warnClosed}
	}
}

// reopened reports whether a Pending grant was closed before: the platform keeps who closed it
// last when the grant is requested again.
func reopened(g webclient.GrantView) bool {
	return g.Status == "Pending" && g.ProcessedBy != ""
}

// closeKey identifies the grant's last decision or close (who and when). A request made again keeps
// it; a later close changes it. "" when the grant was never decided.
func closeKey(g webclient.GrantView) string {
	if g.ProcessedBy == "" {
		return ""
	}
	return g.ProcessedAt + "|" + strings.ToLower(g.ProcessedBy)
}

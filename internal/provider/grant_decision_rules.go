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
	warnApproveNotOffered = "approve not offered"
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

// closeKey identifies the grant's last decision or close (who, when and the comment). A request made
// again keeps it; a later close changes it. "" when the grant was never decided. processedAt has
// one-second resolution, so the comment tells apart two closes by one user within a second.
func closeKey(g webclient.GrantView) string {
	if g.ProcessedBy == "" {
		return ""
	}
	return g.ProcessedAt + "|" + strings.ToLower(g.ProcessedBy) + "|" + g.Comment
}

// withOffer applies the strict default for approve (Design v8, decision 2): the provider approves only
// when the platform offers approve to the signed-in user (the grant's "approve" link). Without it, for
// example with self-approval disabled for the user who requested, nothing is planned and a warning
// says so. The platform's PUT would approve anyway; the provider doesn't go around the platform.
func withOffer(out decisionOutcome, status string, canApprove bool) decisionOutcome {
	if out.Action == actionApprove && !canApprove {
		return decisionOutcome{PlannedStatus: status, Warning: warnApproveNotOffered}
	}
	return out
}

// closedByRequester reports whether the user who closed the grant last also requested it: the
// application team gave up access itself (deleted its grant, or cancelled its request).
func closedByRequester(g webclient.GrantView) bool {
	return g.ProcessedBy != "" && strings.EqualFold(g.ProcessedBy, g.RequestedBy)
}

// adoptClose returns the key of the decision's own last close. Under DENIED a Rejected or Revoked grant
// is what the decision asks for, whoever closed it, so its close counts as the decision's own. A
// decision moved from a rejection or imported has no key of its own; without this, the flip to
// APPROVED would report the old reject as closed outside Terraform. It also repairs a key that wasn't
// recorded because the re-read after the provider's own reject failed.
func adoptClose(decision string, g *webclient.GrantView, ownClose string) string {
	if g != nil && decision == decisionDenied && (g.Status == "Rejected" || g.Status == "Revoked") {
		return closeKey(*g)
	}
	return ownClose
}

// otherApplication reports whether the decision now finds the grant of another application than the
// one it matched first (pinned): the application was deleted and one with the same name was created,
// maybe by another team. Setting application to the new UID (configured) accepts it. A UID can't be
// faked by the other team, unlike who closed a grant.
func otherApplication(pinned, seen, configured string) bool {
	return pinned != "" && seen != "" && !strings.EqualFold(pinned, seen) && !strings.EqualFold(configured, seen)
}

// closedOutside is closed_outside_terraform (Design v8, I7): under APPROVED the grant is Rejected or
// Revoked, the close isn't this decision's own (closeKey differs from ownClose, the key the provider
// stored after its own last reject or revoke) and the application team didn't give up access itself.
// Without a stored key a close counts as outside, the safe side. It only reports: decide never reads it.
func closedOutside(decision, status, closeKey, ownClose string, byRequester bool) bool {
	return decision == decisionApproved && (status == "Rejected" || status == "Revoked") &&
		closeKey != ownClose && !byRequester
}

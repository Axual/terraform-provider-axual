package provider

import "strings"

// The rules of axual_application_access_grant_decision, without Terraform or API types, so they
// can be tested on their own.

const (
	decisionApproved = "APPROVED"
	decisionDenied   = "DENIED"

	statusNotFound = "Not found"

	nextRequestApprove = "approve"
	nextRequestReview  = "review"

	actionNone    = ""
	actionApprove = "approve"
	actionReject  = "reject"
	actionRevoke  = "revoke"
)

// decisionInput is what the rules look at for one decision in one plan or apply.
type decisionInput struct {
	Status   string // the grant's status, or statusNotFound
	Decision string // decisionApproved or decisionDenied
	// Create is true when the block is new in this plan (the PR that adds it reviews the request).
	Create bool
	// DecisionChanged and ReviewChanged compare the plan with the state.
	DecisionChanged bool
	ReviewChanged   bool
	// NextRequest is the stored next_request: nextRequestApprove pre-approves the next request.
	NextRequest string
	// Imported is true for the first plan after an import: the state has no decision yet. Like a
	// create it reviews a Pending request, but it never pre-approves a closed grant, so a bulk import
	// can't re-open access that was rejected or revoked.
	Imported bool
	// ClosedByRequester is true for a request made again after the application team closed the
	// grant itself (it cancelled or revoked, and asked again), not the topic team. See closedByRequester.
	ClosedByRequester bool
}

// decisionOutcome is what the provider does and plans.
type decisionOutcome struct {
	Action        string
	PlannedStatus string
	NextRequest   string // "" means null
	Warning       string // one of the warn* kinds, or ""
}

const (
	warnWaitingForRequest = "waiting"
	warnReopened          = "reopened"
	warnClosed            = "closed"
	warnApprovedAgain     = "approved again"
)

// decide applies the coverage rule:
//
//   - A Pending request is approved when it is covered: the block is created, decision or review
//     changed, or next_request is "approve".
//   - next_request becomes "approve" only through a create or a change of decision to APPROVED
//     while nothing is Pending, and never for a closed grant right after an import. The first
//     request that arrives uses it up.
//   - A request made again after the application team closed the grant itself is approved again
//     under APPROVED. After a reject or revoke by the topic side it waits for a review.
//   - DENIED rejects a Pending request and revokes an Approved one, without any coverage.
func decide(in decisionInput) decisionOutcome {
	if in.Decision == decisionDenied {
		switch in.Status {
		case "Pending":
			return decisionOutcome{Action: actionReject, PlannedStatus: "Rejected"}
		case "Approved":
			return decisionOutcome{Action: actionRevoke, PlannedStatus: "Revoked"}
		}
		return decisionOutcome{PlannedStatus: in.Status}
	}

	preApprove := in.Create || in.DecisionChanged
	switch in.Status {
	case "Pending":
		if in.Create || in.DecisionChanged || in.ReviewChanged || in.NextRequest == nextRequestApprove {
			return decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview}
		}
		if in.ClosedByRequester {
			// The application team's own cancel or revoke doesn't need the topic team again.
			return decisionOutcome{Action: actionApprove, PlannedStatus: "Approved", NextRequest: nextRequestReview, Warning: warnApprovedAgain}
		}
		return decisionOutcome{PlannedStatus: "Pending", NextRequest: nextRequestReview, Warning: warnReopened}
	case "Approved":
		return decisionOutcome{PlannedStatus: "Approved", NextRequest: nextRequestReview}
	case statusNotFound:
		return decisionOutcome{PlannedStatus: in.Status, NextRequest: keepOrApprove(in.NextRequest, preApprove), Warning: warnWaitingForRequest}
	default: // Rejected, Revoked, Cancelled
		if in.Imported {
			preApprove = false
		}
		return decisionOutcome{PlannedStatus: in.Status, NextRequest: keepOrApprove(in.NextRequest, preApprove), Warning: warnClosed}
	}
}

func keepOrApprove(current string, preApprove bool) string {
	if preApprove || current == nextRequestApprove {
		return nextRequestApprove
	}
	return nextRequestReview
}

// closedByRequester reports whether a Pending grant was reopened after the application team itself
// closed it: the last cancel or revoke (processedBy) was done by the user who asked again
// (requestedBy), and that user isn't the signed-in user (the topic side's own identity). With one
// account for both sides, or when anything is unknown, it is false, so the request waits for a review.
func closedByRequester(status, processedBy, requestedBy, me string) bool {
	return status == "Pending" && processedBy != "" && me != "" &&
		strings.EqualFold(processedBy, requestedBy) && !strings.EqualFold(processedBy, me)
}

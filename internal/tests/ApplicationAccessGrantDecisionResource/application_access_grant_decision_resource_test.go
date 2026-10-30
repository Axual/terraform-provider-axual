package ApplicationAccessGrantDecisionResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	decision = "axual_application_access_grant_decision.tf-test-decision"
	grant    = "axual_application_access_grant.tf-test-grant"
)

func config(files ...string) string {
	out := GetProvider() + GetFile("base.tf")
	for _, f := range files {
		out += GetFile(f)
	}
	return out
}

// TestApplicationAccessGrantDecision walks the multi-repository flow in one state, with one account
// for both sides as an acceptance test has:
//   - the decision exists before the request, and the request is approved by the next apply;
//   - DENIED revokes; a request made again is rejected again;
//   - DENIED to APPROVED before the request: the next request is approved;
//   - removing the grant (revoke) and adding it back is approved again under APPROVED;
//   - the decision imports by its composite id.
func TestApplicationAccessGrantDecision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			// 1. Decision before the request: create works, nothing to approve yet.
			{
				Config: config("decision_approved.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Not found"),
					resource.TestCheckNoResourceAttr(decision, "grant_id"),
					resource.TestMatchResourceAttr(decision, "id", regexp.MustCompile(`^[0-9a-f]{32}/[0-9a-f]{32}/CONSUMER/tf-test-decision-app$`)),
				),
			},
			// 2. The application team requests. The decision only sees it on the next refresh.
			{
				Config:             config("grant.tf", "decision_approved.tf"),
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr(grant, "status", "Pending"),
			},
			// 3. The next apply approves the request; the id doesn't change.
			{
				Config: config("grant.tf", "decision_approved.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
					resource.TestCheckResourceAttrPair(decision, "grant_id", grant, "id"),
					resource.TestMatchResourceAttr(decision, "id", regexp.MustCompile(`/CONSUMER/tf-test-decision-app$`)),
				),
			},
			// 4. DENIED with a reason revokes.
			{
				Config: config("grant.tf", "decision_denied.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Revoked"),
			},
			// 5. The application team sees the decision.
			{
				Config: config("grant.tf", "decision_denied.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(grant, "status", "Revoked"),
					resource.TestCheckResourceAttr(grant, "decision_reason", "Not allowed, see INC-1234"),
					resource.TestCheckResourceAttrSet(grant, "decided_at"),
				),
			},
			// 6. The application team asks again; DENIED rejects the new request at the next apply.
			{
				Config:             config("grant_again1.tf", "decision_denied.tf"),
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr(grant, "status", "Pending"),
			},
			{
				Config: config("grant_again1.tf", "decision_denied.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Rejected"),
			},
			// 7. Topic team first: DENIED to APPROVED, nothing to approve yet.
			{
				Config: config("grant_again1.tf", "decision_approved.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Rejected"),
			},
			// 8. Then the application team asks again, and the next apply approves it.
			{
				Config:             config("grant_again2.tf", "decision_approved.tf"),
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr(grant, "status", "Pending"),
			},
			{
				Config: config("grant_again2.tf", "decision_approved.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
			},
			// 9. The grant is removed (revoked) and added back. The standing APPROVED approves the new
			//    request at the next apply.
			{
				Config: config("decision_approved.tf"),
			},
			{
				Config:             config("grant_again2.tf", "decision_approved.tf"),
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr(grant, "status", "Pending"),
			},
			{
				Config: config("grant_again2.tf", "decision_approved.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
			},
			// 10. Import by the composite id.
			{
				ResourceName:            decision,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"decision", "reason"},
			},
			{
				Destroy: true,
				Config:  config("grant_again2.tf", "decision_approved.tf"),
			},
		},
	})
}

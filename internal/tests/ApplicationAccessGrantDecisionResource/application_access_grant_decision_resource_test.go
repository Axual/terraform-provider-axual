package ApplicationAccessGrantDecisionResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const decision = "axual_application_access_grant_decision.tf-test-decision"

// TestApplicationAccessGrantDecision walks the multi-repository flow in one state:
// the topic team's decision exists before the request, the request is approved by the next apply,
// DENIED revokes it, and the decision imports by its composite id.
func TestApplicationAccessGrantDecision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			// 1. Decision before the request: create works, nothing to approve yet.
			{
				Config: GetProvider() + GetFile("base.tf") + GetFile("decision_approved.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Not found"),
					resource.TestCheckResourceAttr(decision, "next_request", "approve"),
					resource.TestCheckNoResourceAttr(decision, "grant_id"),
					resource.TestMatchResourceAttr(decision, "id", regexp.MustCompile(`^[0-9a-f]{32}/[0-9a-f]{32}/CONSUMER/tf-test-decision-app$`)),
				),
			},
			// 2. The application team requests. The decision only sees it on the next refresh.
			{
				Config:             GetProvider() + GetFile("base.tf") + GetFile("grant.tf") + GetFile("decision_approved.tf"),
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-grant", "status", "Pending"),
			},
			// 3. The next apply approves the pre-approved request; the id doesn't change.
			{
				Config: GetProvider() + GetFile("base.tf") + GetFile("grant.tf") + GetFile("decision_approved.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
					resource.TestCheckResourceAttr(decision, "next_request", "review"),
					resource.TestCheckResourceAttrPair(decision, "grant_id", "axual_application_access_grant.tf-test-grant", "id"),
					resource.TestMatchResourceAttr(decision, "id", regexp.MustCompile(`/CONSUMER/tf-test-decision-app$`)),
				),
			},
			// 4. DENIED with a reason revokes.
			{
				Config: GetProvider() + GetFile("base.tf") + GetFile("grant.tf") + GetFile("decision_denied.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Revoked"),
					resource.TestCheckNoResourceAttr(decision, "next_request"),
				),
			},
			// 5. The application team sees the decision.
			{
				Config: GetProvider() + GetFile("base.tf") + GetFile("grant.tf") + GetFile("decision_denied.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-grant", "status", "Revoked"),
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-grant", "decision_reason", "Not allowed, see INC-1234"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-grant", "decided_at"),
				),
			},
			// 6. Import by the composite id.
			{
				ResourceName:            decision,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"decision", "reason", "review", "next_request", "grant"},
			},
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("base.tf") + GetFile("grant.tf") + GetFile("decision_denied.tf"),
			},
		},
	})
}

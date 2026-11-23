package ApplicationAccessGrantDecisionResource

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	decision = "axual_application_access_grant_decision.tf-test-decision"
	grant    = "axual_application_access_grant.tf-test-grant"
	grants   = "data.axual_application_access_grants.tf-test-grants"
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
			// 5. The application team sees the decision. decided_by of the grant comes from
			//    GET /application_access_grants/{id}, decided_by of the grants data source from the
			//    grant search (processedBy), so they are the same only when the search has the email.
			{
				Config: config("grant.tf", "decision_denied.tf", "grants_data_source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(grant, "status", "Revoked"),
					resource.TestCheckResourceAttr(grant, "decision_reason", "Not allowed, see INC-1234"),
					resource.TestCheckResourceAttrSet(grant, "decided_at"),
					resource.TestCheckResourceAttrSet(grant, "decided_by"),
					resource.TestCheckResourceAttr(grants, "grants.#", "1"),
					resource.TestCheckResourceAttrPair(grants, "grants.0.id", grant, "id"),
					resource.TestCheckResourceAttrPair(grants, "grants.0.decided_by", grant, "decided_by"),
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

// TestApplicationAccessGrantDecisionFromApproval moves an approval to a decision. Deleting the approval
// and adding the decision without moved fails the plan: the removal would revoke the grant that the
// decision approves. With moved the grant stays Approved.
func TestApplicationAccessGrantDecisionFromApproval(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: config("grant.tf", "approval.tf"),
				Check:  resource.TestCheckResourceAttrPair("axual_application_access_grant_approval.tf-test-decision", "application_access_grant", grant, "id"),
			},
			// The grant sees the approval on the next refresh.
			{
				Config: config("grant.tf", "approval.tf"),
				Check:  resource.TestCheckResourceAttr(grant, "status", "Approved"),
			},
			{
				Config:      config("grant.tf", "decision_approved.tf"),
				ExpectError: regexp.MustCompile(`is removed and another one approves it`),
			},
			{
				Config: config("grant.tf", "decision_approved.tf", "moved_from_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
					resource.TestCheckResourceAttr(grant, "status", "Approved"),
				),
			},
			{
				Destroy: true,
				Config:  config("grant.tf", "decision_approved.tf", "moved_from_approval.tf"),
			},
		},
	})
}

// idNamesApplication checks that the decision's id ends with the UID of the application, so it follows
// a replaced application.
func idNamesApplication(s *terraform.State) error {
	d, app := s.RootModule().Resources[decision], s.RootModule().Resources["axual_application.tf-test-app"]
	if d == nil || app == nil {
		return fmt.Errorf("decision or application not in the state")
	}
	if !strings.HasSuffix(d.Primary.ID, "/"+app.Primary.ID) {
		return fmt.Errorf("decision id %s doesn't end with the application UID %s", d.Primary.ID, app.Primary.ID)
	}
	return nil
}

// TestApplicationAccessGrantDecisionApplicationReplaced replaces the application (and so its grant) of a
// decision that names it by UID. The UID is unknown in the plan, so the decision is planned as an update,
// and it must stay one when the apply knows the new UID: it approves the new grant in the same apply,
// and id follows the new application. Before, the apply failed with "Provider produced inconsistent
// final plan".
func TestApplicationAccessGrantDecisionApplicationReplaced(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: config("grant.tf", "decision_by_uid.tf"),
				Check:  resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
			},
			{
				Config: config("grant.tf", "decision_by_uid.tf"),
				Taint:  []string{"axual_application.tf-test-app", grant},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axual_application.tf-test-app", plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction(decision, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(decision, "current_status", "Approved"),
					resource.TestCheckResourceAttrPair(decision, "application_uid", "axual_application.tf-test-app", "id"),
					resource.TestCheckResourceAttrPair(decision, "grant_id", grant, "id"),
					idNamesApplication,
				),
			},
			{
				Destroy: true,
				Config:  config("grant.tf", "decision_by_uid.tf"),
			},
		},
	})
}

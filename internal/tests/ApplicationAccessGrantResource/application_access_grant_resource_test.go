package ApplicationAccessGrantResource

import (
	"fmt"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// =============================================================================
// Auto Environment Tests
// =============================================================================

// TestApplicationAccessGrant_AutoEnvironment tests grant in Auto environment
// where grants are automatically approved upon creation.
//
// Why approval resource is needed:
//   - In Auto environment, the grant is auto-approved (status = "Approved")
//   - Approved grants cannot be deleted directly - they must be REVOKED first
//   - The approval resource's Delete function triggers revocation
//   - Without approval resource, cleanup would fail with:
//     "Application Access Grant cannot be cancelled. Please Revoke first."
//
// Cleanup order:
//  1. Delete approval resource → calls RevokeOrDenyGrant() → grant status becomes "Revoked"
//  2. Delete grant resource → calls CancelGrant() → works because grant is now "Revoked"
func TestApplicationAccessGrant_AutoEnvironment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources - grant should be auto-approved
			// In Auto environment, grant.Create() returns status = "Approved" immediately
			// No state sync issue here(like in Stream Owner env) because approval happens during grant creation, not after
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_auto.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant attributes
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
					// Verify references are set
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "application"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "topic"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "environment"),
					// Verify environment is Auto
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Auto"),
					// Verify approval resource created (needed for revoke during cleanup)
					resource.TestCheckResourceAttrSet("axual_application_access_grant_approval.tf-test-application-access-grant-approval", "application_access_grant"),
				),
			},
			// Step 2: Import - should work without ImportStateVerifyIgnore
			// No refresh step needed here because in Auto environment, the grant is
			// already "Approved" when created - there's no state sync issue like in Stream Owner env
			{
				ResourceName:      "axual_application_access_grant.tf-test-application-access-grant",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Cleanup
			// Destroy order: approval.Delete() revokes grant, then grant.Delete() cancels it
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_auto.tf"),
			},
		},
	})
}

// =============================================================================
// Topic Owner (Stream Owner) Environment Tests
// =============================================================================

// TestApplicationAccessGrant_TopicOwnerApproval tests the full lifecycle in Topic Owner environment:
// Create (Pending) -> Approval -> Refresh -> Import -> Cleanup
//
// KNOWN ISSUE - State Sync Problem:
// When the approval resource is created, it calls the API to approve the grant.
// This changes the grant's status in the API from "Pending" to "Approved".
// However, the grant resource's Terraform state is NOT updated - it still shows "Pending".
//
// Timeline:
//  1. grant.Create() → API returns status="Pending" → state saved as "Pending"
//  2. approval.Create() → calls ApproveGrant() API → API changes grant to "Approved"
//     → BUT grant's Terraform state is NOT updated (still "Pending")
//  3. After apply: API has "Approved", Terraform state has "Pending" (STATE DRIFT!)
//
// This is why we need a REFRESH STEP (Step 2) before import:
//   - Re-applying the same config triggers terraform's refresh phase
//   - Refresh phase calls grant.Read() which fetches current status from API
//   - State is updated from "Pending" to "Approved"
//   - Now import verification will pass (state matches API)
//
// See: docs/bug-grant-state-not-updated-after-approval.md for full analysis
func TestApplicationAccessGrant_TopicOwnerApproval(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources
			// Grant is created with status="Pending", then approval resource approves it.
			// After this step:
			//   - API: grant status = "Approved"
			//   - Terraform state: grant status = "Pending" (STALE!)
			// We intentionally don't check status here because it would show "Pending"
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant attributes (but NOT status - it's stale)
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					// Verify references are set
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "application"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "topic"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "environment"),
					// Verify environment is Topic owner (Stream owner)
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
					// Verify approval resource created
					resource.TestCheckResourceAttrSet("axual_application_access_grant_approval.tf-test-application-access-grant-approval", "application_access_grant"),
				),
			},
			// Step 2: REFRESH STEP - Re-apply same config to sync state with API
			//
			// How this works:
			//   1. Terraform starts "apply" with same config
			//   2. REFRESH PHASE runs first - calls Read() on all resources
			//   3. grant.Read() fetches current status from API ("Approved")
			//   4. Terraform state is updated: status "Pending" → "Approved"
			//   5. PLAN PHASE - no changes needed (config matches refreshed state)
			//   6. APPLY PHASE - nothing to apply
			//
			// After this step: both API and Terraform state have status="Approved"
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					// NOW we can verify status is "Approved" - state has been synced
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
				),
			},
			// Step 3: Import verification
			// This works without ImportStateVerifyIgnore because Step 2 synced the state.
			// Import process:
			//   1. Save current state (status="Approved")
			//   2. Remove resource from state
			//   3. Import using ID - calls Read() - fetches status="Approved"
			//   4. Compare: "Approved" == "Approved" ✓
			{
				ResourceName:      "axual_application_access_grant.tf-test-application-access-grant",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Cleanup
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
			},
		},
	})
}

// TestApplicationAccessGrant_TopicOwnerRejection tests grant rejection flow
//
// Same state sync issue as approval:
// After rejection.Create(), the grant's status changes to "Rejected" in API
// but Terraform state still shows "Pending" until refreshed.
func TestApplicationAccessGrant_TopicOwnerRejection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources - grant is created then rejected
			// After this step: API has "Rejected", state has "Pending" (stale)
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant attributes (but NOT status - it's stale)
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					// Verify references are set
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "application"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "topic"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "environment"),
					// Verify environment is Topic owner (Stream owner)
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
					// Verify rejection resource created
					resource.TestCheckResourceAttrSet("axual_application_access_grant_rejection.tf-test-application-access-grant-rejection", "application_access_grant"),
				),
			},
			// Step 2: REFRESH STEP - sync state with API, verify status is "Rejected"
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
				Check: resource.ComposeTestCheckFunc(
					// After refresh, status should be "Rejected"
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Rejected"),
				),
			},
			// Step 3: Cleanup
			// Rejected grants can be cancelled directly (no revoke needed)
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
			},
		},
	})
}

// TestApplicationAccessGrant_PendingImport tests import of a grant that stays in Pending status
//
// This test does NOT have the state sync issue because:
//   - No approval/rejection resource is created
//   - Grant stays in "Pending" status
//   - State matches API (both "Pending")
//   - No refresh step needed
//
// This demonstrates that the state sync issue only occurs when approval/rejection
// resources modify the grant's status after creation.
func TestApplicationAccessGrant_PendingImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources WITHOUT approval - grant stays Pending
			// No state sync issue here - status is "Pending" in both state and API
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_pending.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Can check status here - no state drift
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Pending"),
					// Verify references are set
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "application"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "topic"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "environment"),
					// Verify environment is Topic owner (Stream owner)
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
				),
			},
			// Step 2: Import - works without refresh step or ImportStateVerifyIgnore
			// because status is "Pending" in both state and API (no drift)
			{
				ResourceName:      "axual_application_access_grant.tf-test-application-access-grant",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Cleanup
			// Pending grants can be cancelled directly (no revoke needed)
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_pending.tf"),
			},
		},
	})
}

// =============================================================================
// Approval Resource Import Tests
// =============================================================================

// TestApplicationAccessGrantApproval_Import tests import of the approval resource
//
// This verifies that:
//  1. The approval resource can be imported using the grant UID
//  2. After import, the approval resource's state matches the API
//  3. The Read function correctly handles the "Approved" status
//
// Note: This requires a refresh step before import due to the state sync issue
// (grant status shows "Pending" in state while API has "Approved")
func TestApplicationAccessGrantApproval_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources - grant is created then approved
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify approval resource created
					resource.TestCheckResourceAttrSet("axual_application_access_grant_approval.tf-test-application-access-grant-approval", "application_access_grant"),
					// Verify environment is Stream owner (requires manual approval)
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
				),
			},
			// Step 2: REFRESH STEP - sync state with API
			// This is needed because the approval resource's Create changes the grant status
			// from "Pending" to "Approved", but the grant resource's state is not updated
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant is now Approved after refresh
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
				),
			},
			// Step 3: Import the APPROVAL resource
			// This tests the approval resource's Read function and ImportState
			// Notes:
			// - ImportStateIdFunc: needed because the resource doesn't have an 'id' attribute
			// - ImportStateVerifyIdentifierAttribute: tells verify to use 'application_access_grant' as identifier
			{
				ResourceName:                         "axual_application_access_grant_approval.tf-test-application-access-grant-approval",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "application_access_grant",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["axual_application_access_grant_approval.tf-test-application-access-grant-approval"]
					if !ok {
						return "", fmt.Errorf("Resource not found in state")
					}
					return rs.Primary.Attributes["application_access_grant"], nil
				},
			},
			// Step 4: Cleanup
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_stream_owner_approval.tf"),
			},
		},
	})
}

// =============================================================================
// Rejection Resource Import Tests
// =============================================================================

// TestApplicationAccessGrantRejection_Import tests import of the rejection resource
//
// This verifies that:
//  1. The rejection resource can be imported using the grant UID
//  2. After import, the rejection resource's state matches the API
//  3. The Read function correctly handles the "Rejected" status
//
// Note: The 'reason' attribute is not returned by the API, so after import
// it will be null. This is expected behavior (see docs/rejection-import-reason-workarounds.md)
func TestApplicationAccessGrantRejection_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources - grant is created then rejected
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify rejection resource created
					resource.TestCheckResourceAttrSet("axual_application_access_grant_rejection.tf-test-application-access-grant-rejection", "application_access_grant"),
					// Verify environment is Stream owner
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Stream owner"),
				),
			},
			// Step 2: REFRESH STEP - sync state with API
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant is now Rejected after refresh
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Rejected"),
				),
			},
			// Step 3: Import the REJECTION resource
			// This tests the rejection resource's Read function and ImportState
			// Notes:
			// - ImportStateIdFunc: needed because the resource doesn't have an 'id' attribute
			// - ImportStateVerifyIdentifierAttribute: tells verify to use 'application_access_grant' as identifier
			{
				ResourceName:                         "axual_application_access_grant_rejection.tf-test-application-access-grant-rejection",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "application_access_grant",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["axual_application_access_grant_rejection.tf-test-application-access-grant-rejection"]
					if !ok {
						return "", fmt.Errorf("Resource not found in state")
					}
					return rs.Primary.Attributes["application_access_grant"], nil
				},
			},
			// Step 4: Cleanup
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection.tf"),
			},
		},
	})
}

// TestApplicationAccessGrantRejection_ImportWithReason tests import with reason attribute
//
// The 'reason' attribute is not returned by the API. After import, it will be null.
// This test uses ImportStateVerifyIgnore to handle this known limitation.
func TestApplicationAccessGrantRejection_ImportWithReason(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources with reason
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection_with_reason.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify rejection resource created with reason
					resource.TestCheckResourceAttrSet("axual_application_access_grant_rejection.tf-test-application-access-grant-rejection", "application_access_grant"),
					resource.TestCheckResourceAttr("axual_application_access_grant_rejection.tf-test-application-access-grant-rejection", "reason", "Not authorized for this environment"),
				),
			},
			// Step 2: REFRESH STEP
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection_with_reason.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Rejected"),
				),
			},
			// Step 3: Import - must ignore 'reason' as API doesn't return it
			// Notes:
			// - ImportStateIdFunc: needed because the resource doesn't have an 'id' attribute
			// - ImportStateVerifyIdentifierAttribute: tells verify to use 'application_access_grant' as identifier
			// - ImportStateVerifyIgnore: 'reason' is not returned by API
			{
				ResourceName:                         "axual_application_access_grant_rejection.tf-test-application-access-grant-rejection",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "application_access_grant",
				ImportStateVerifyIgnore:              []string{"reason"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["axual_application_access_grant_rejection.tf-test-application-access-grant-rejection"]
					if !ok {
						return "", fmt.Errorf("Resource not found in state")
					}
					return rs.Primary.Attributes["application_access_grant"], nil
				},
			},
			// Step 4: Cleanup
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_stream_owner_rejection_with_reason.tf"),
			},
		},
	})
}

// =============================================================================
// Grant Auto-Revoke Tests
// =============================================================================

// TestApplicationAccessGrant_AutoRevokeOnDelete tests that an approved grant can be
// deleted without a separate approval resource.
//
// Previously, deleting an approved grant would fail with:
//   "Application Access Grant cannot be cancelled. Please Revoke first."
//
// With the fix to the grant Delete function, approved grants are automatically
// revoked before being removed from state. This enables:
//  1. Terraform destroy to work after import (when dependency is lost)
//  2. Simpler configurations where approval resource is not strictly needed for cleanup
//
// This test creates a grant in Auto environment (auto-approved) WITHOUT an approval
// resource, then verifies that destroy succeeds.
func TestApplicationAccessGrant_AutoRevokeOnDelete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create grant in Auto environment - it will be auto-approved
			// Note: NO approval resource in this config
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_auto_no_approval.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify grant is auto-approved
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "CONSUMER"),
					// Verify environment is Auto
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Auto"),
				),
			},
			// Step 2: Destroy - this should succeed because grant.Delete() now auto-revokes
			// Previously this would fail with "Please Revoke first"
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_auto_no_approval.tf"),
			},
		},
	})
}

// =============================================================================
// Access Type Tests
// =============================================================================

// TestApplicationAccessGrant_ProducerAccess tests PRODUCER access type
//
// Uses Auto environment so grant is auto-approved.
// Same cleanup requirement as TestApplicationAccessGrant_AutoEnvironment:
// approval resource needed for revoke during cleanup.
func TestApplicationAccessGrant_ProducerAccess(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			// Step 1: Create resources with PRODUCER access type
			// In Auto environment, grant is auto-approved - no state sync issue
			{
				Config: GetProvider() + GetFile("axual_application_access_grant_producer.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Verify access type is PRODUCER
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "access_type", "PRODUCER"),
					// In Auto environment, should be auto-approved
					resource.TestCheckResourceAttr("axual_application_access_grant.tf-test-application-access-grant", "status", "Approved"),
					// Verify references are set
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "id"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "application"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "topic"),
					resource.TestCheckResourceAttrSet("axual_application_access_grant.tf-test-application-access-grant", "environment"),
					// Verify environment
					resource.TestCheckResourceAttr("axual_environment.tf-test-env", "authorization_issuer", "Auto"),
					// Verify approval resource created (needed for revoke during cleanup)
					resource.TestCheckResourceAttrSet("axual_application_access_grant_approval.tf-test-application-access-grant-approval", "application_access_grant"),
				),
			},
			// Step 2: Import - no refresh needed (Auto env = no state sync issue)
			{
				ResourceName:      "axual_application_access_grant.tf-test-application-access-grant",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Cleanup (approval.Delete() revokes, then grant.Delete() cancels)
			{
				Destroy: true,
				Config:  GetProvider() + GetFile("axual_application_access_grant_producer.tf"),
			},
		},
	})
}

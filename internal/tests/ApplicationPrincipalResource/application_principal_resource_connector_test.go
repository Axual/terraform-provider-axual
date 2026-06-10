package ApplicationPrincipalResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestApplicationPrincipalConnectorResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", CertPath("generic_application_1.key")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal", "active"),
					// Created without active -> inactive in the API.
					CheckPrincipalActiveInAPI("axual_application_principal.connector_axual_application_principal", false),
				),
			},
			{
				// Rotate the certificate resource: a new principal is uploaded and the old one deleted.
				// active is omitted and the principal being replaced is inactive in the API, so the
				// rotation inherits the inactive status — the rotated principal stays inactive.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", CertPath("generic_application_2.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", CertPath("generic_application_2.key")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal", "active"),
					CheckPrincipalActiveInAPI("axual_application_principal.connector_axual_application_principal", false),
				),
			},
			{
				// Replace the certificate resource: first new principal (as new resource) is uploaded with `active=true`, then old resource one deleted
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_removed.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "private_key", CertPath("generic_application_1.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_new", "active", "true"),
				),
			},
			{
				// Replace the certificate resource: first 2 new principals (as new resources) are uploaded one with `active=true`, then old resource one deleted
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", CertPath("generic_application_4.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", CertPath("generic_application_4.key")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", CertPath("generic_application_2.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", CertPath("generic_application_2.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Add _last (active=true): activates _last via atomic swap, deactivating _two in API.
				// _two's resource retains active=true (write-only intent, not refreshed from API) even though
				// _two is now API-inactive.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_inactive.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", CertPath("generic_application_2.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", CertPath("generic_application_2.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "private_key", CertPath("generic_application_1.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_last", "active", "true"),
				),
			},
			{
				ResourceName:            "axual_application_principal.connector_axual_application_principal_last",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"private_key", "active", "principal"},
			},
			{
				// Verify that deleting an unused principal returns no error
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_inactive.tf",
				),
			},
		},
	})
}

func TestApplicationPrincipalConnectorDeleteActiveFails(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Setup: create one principal with active=true and a running deployment so deletion is blocked
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_fails_setup.tf",
					"axual_application_principal_connector_added_removed.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "private_key", CertPath("generic_application_1.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_new", "active", "true"),
				),
			},
			{
				// Attempt to delete the active principal without first activating another — expect error because connector is running
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_fails_setup.tf",
					"axual_application_principal_connector_two_added_removed_fail.tf",
				),
				ExpectError: regexp.MustCompile("Unable to delete.*principal"),
			},
			{
				// Step 3a: activate _two (atomic swap deactivates _new in API) while _new is still present.
				// Cert-unchanged update on _two (active null→true) takes the fast path — no rotation, no duplicate POST.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_fails_setup.tf",
					"axual_application_principal_connector_added_removed.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", CertPath("generic_application_4.cer")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", CertPath("generic_application_2.cer")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Step 3b: drop _new — now API-inactive (atomic-swapped in 3a), so deletion succeeds, even with deployment running.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_fails_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", CertPath("generic_application_4.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", CertPath("generic_application_4.key")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", CertPath("generic_application_2.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", CertPath("generic_application_2.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Teardown prelude: remove the deployment/grant so principals are no longer attached to a running connector.
				// Without this, the final Destroy step races principal-delete against deployment-delete and hits the deployment guard.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
			},
		},
	})
}

// TestApplicationPrincipalConnectorRotationInheritsActivation verifies that a cert rotation
// inherits the LIVE API activation status of the principal being replaced when there is no fresh
// active=true transition. A principal created active=true stays active across a rotation even
// though `active` is omitted on the rotating apply (state mirrors config, so it shows no attr,
// but the API keeps it active).
func TestApplicationPrincipalConnectorRotationInheritsActivation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Create active=true -> active in the API.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_active_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", CertPath("generic_application_1.cer")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal", "active", "true"),
					CheckPrincipalActiveInAPI("axual_application_principal.connector_axual_application_principal", true),
				),
			},
			{
				// Rotate cert+key with active omitted. No fresh transition (state had no active=true
				// either, since rotation creates a new principal), so activation is inherited from the
				// old principal's live API status (active) -> rotated principal stays active.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_active_rotated.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", CertPath("generic_application_2.cer")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal", "active"),
					CheckPrincipalActiveInAPI("axual_application_principal.connector_axual_application_principal", true),
				),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_active_rotated.tf",
				),
			},
		},
	})
}

// TestApplicationPrincipalConnectorRotationPersistentActiveDefersToAPI covers the "deactivated
// outside this resource" scenario: `p` carries a persistent active=true while it has been
// deactivated in the API (by activating `other`). Rotating `p` must NOT reactivate it — with no
// fresh active transition, the rotation inherits the live API status (inactive).
func TestApplicationPrincipalConnectorRotationPersistentActiveDefersToAPI(t *testing.T) {
	pName := "axual_application_principal.connector_axual_application_principal"
	otherName := "axual_application_principal.connector_axual_application_principal_other"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Setup: `p` active=true, `other` inactive.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_setup.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckPrincipalActiveInAPI(pName, true),
					CheckPrincipalActiveInAPI(otherName, false),
				),
			},
			{
				// Activate `other` -> atomic swap deactivates `p` in the API. `p`'s state keeps
				// active=true (write-only intent, not refreshed).
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_swap.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckPrincipalActiveInAPI(pName, false),
					CheckPrincipalActiveInAPI(otherName, true),
				),
			},
			{
				// Rotate `p` (cert gen1->gen2) keeping active=true. state.Active was true the whole
				// time => NO fresh transition => inherit `p` live API status (inactive) => rotated `p`
				// stays inactive; `other` keeps serving. (`other` drops active here: a cert-unchanged
				// in-place toggle with no API call, so it remains active in the API.)
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_rotate.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile(pName, "principal", CertPath("generic_application_2.cer")),
					CheckPrincipalActiveInAPI(pName, false),
					CheckPrincipalActiveInAPI(otherName, true),
				),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_rotate.tf",
				),
			},
		},
	})
}

// TestApplicationPrincipalConnectorRotationFreshFlipActivates covers the counterpart: when the user
// removes active and re-adds it in the same apply as the rotation, that config transition
// (false/null -> true) IS honored — the rotated principal is activated even though it was inactive
// in the API. The same final fixture as the persistent test above; only the intervening toggle differs.
func TestApplicationPrincipalConnectorRotationFreshFlipActivates(t *testing.T) {
	pName := "axual_application_principal.connector_axual_application_principal"
	otherName := "axual_application_principal.connector_axual_application_principal_other"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Setup: `p` active=true, `other` inactive.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_setup.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckPrincipalActiveInAPI(pName, true),
					CheckPrincipalActiveInAPI(otherName, false),
				),
			},
			{
				// Activate `other` -> atomic swap deactivates `p` in the API.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_swap.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckPrincipalActiveInAPI(pName, false),
					CheckPrincipalActiveInAPI(otherName, true),
				),
			},
			{
				// Prelude: remove active=true from `p` (cert unchanged in-place toggle). state.Active
				// goes null; no API call, so `p` stays inactive and `other` stays active.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_toggle.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(pName, "active"),
					CheckPrincipalActiveInAPI(pName, false),
					CheckPrincipalActiveInAPI(otherName, true),
				),
			},
			{
				// Fresh flip: rotate `p` (gen1->gen2) AND add active=true in the same apply.
				// state.Active was null => boolTrue(plan.Active) && !boolTrue(state.Active) == true
				// => fresh transition => activate rotated `p` (atomic swap deactivates `other`).
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_rotate.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile(pName, "principal", CertPath("generic_application_2.cer")),
					resource.TestCheckResourceAttr(pName, "active", "true"),
					CheckPrincipalActiveInAPI(pName, true),
					CheckPrincipalActiveInAPI(otherName, false),
				),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_freshflip_rotate.tf",
				),
			},
		},
	})
}

// TestApplicationPrincipalConnectorKeyTrailingWhitespaceNoDiff guards against a misleading plan
// when only trailing whitespace (e.g. a trailing newline that file() preserves) differs on
// principal or private_key. principal was already protected; private_key was compared exactly, so
// a trailing newline marked the cert as rotating and the plan showed `id -> (known after apply)`,
// implying a rotation that apply would not actually perform. Both fields now suppress
// whitespace-only diffs, so re-applying the same cert with extra trailing newlines is an empty plan.
func TestApplicationPrincipalConnectorKeyTrailingWhitespaceNoDiff(t *testing.T) {
	const name = "axual_application_principal.connector_axual_application_principal"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Create with the plain cert/key (as file() returns them).
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_initial.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile(name, "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile(name, "private_key", CertPath("generic_application_1.key")),
				),
			},
			{
				// Same cert/key but with extra trailing newlines appended to both fields. This must NOT
				// be seen as a rotation: the plan must be empty (no `id -> known after apply`).
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_key_trailing_ws.tf",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_key_trailing_ws.tf",
				),
			},
		},
	})
}

func TestApplicationPrincipalConnectorAtomicSwap(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
		ExternalProviders:        GetProviderConfig(t).ExternalProviders,

		Steps: []resource.TestStep{
			{
				// Setup: two principals, _two active
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", CertPath("generic_application_4.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", CertPath("generic_application_4.key")),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", CertPath("generic_application_2.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", CertPath("generic_application_2.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Atomic swap: _last created with active=true atomically deactivates _two in API.
				// No error: the API swaps the active principal rather than rejecting the second activation.
				// _two's state retains active=true (write-only intent, not refreshed from API).
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_active.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "principal", CertPath("generic_application_1.cer")),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "private_key", CertPath("generic_application_1.key")),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_last", "active", "true"),
				),
			},
			{
				// Transition to _last only: _one and _two are API-inactive so they can be deleted cleanly.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_last_only.tf",
				),
			},
			{
				Destroy: true,
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_last_only.tf",
				),
			},
		},
	})
}

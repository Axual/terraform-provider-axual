package ApplicationPrincipalResource

import (
	"regexp"
	"testing"

	. "axual.com/terraform-provider-axual/internal/tests"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", "certs/generic_application_1.key"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal", "active"),
				),
			},
			{
				// Rotate the certificate resource: new principal is uploaded, activated, and old one deleted
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_replaced.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal", "private_key", "certs/generic_application_2.key"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal", "active"),
				),
			},
			{
				// Replace the certificate resource: first new principal (as new resource) is uploaded with `active=true`, then old resource one deleted
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_removed.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "private_key", "certs/generic_application_1.key"),
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", "certs/generic_application_4.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", "certs/generic_application_4.key"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", "certs/generic_application_2.key"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Add _last (active=true): activates _last via atomic swap, deactivating _two in API.
				// _two's state retains active=true (write-only intent, not refreshed from API) even though
				// _two is now API-inactive.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_added_inactive.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", "certs/generic_application_2.key"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "private_key", "certs/generic_application_1.key"),
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "private_key", "certs/generic_application_1.key"),
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_new", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", "certs/generic_application_4.cer"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", "certs/generic_application_2.cer"),
					resource.TestCheckResourceAttr("axual_application_principal.connector_axual_application_principal_two", "active", "true"),
				),
			},
			{
				// Step 3b: drop _new — now API-inactive (atomic-swapped in 3a), so deletion succeeds even with deployment running.
				Config: GetProvider() + GetFile(
					"axual_application_principal_connector_setup.tf",
					"axual_application_principal_connector_delete_fails_setup.tf",
					"axual_application_principal_connector_two_added_removed_pass.tf",
				),
				Check: resource.ComposeTestCheckFunc(
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", "certs/generic_application_4.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", "certs/generic_application_4.key"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", "certs/generic_application_2.key"),
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "principal", "certs/generic_application_4.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_one", "private_key", "certs/generic_application_4.key"),
					resource.TestCheckNoResourceAttr("axual_application_principal.connector_axual_application_principal_one", "active"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "principal", "certs/generic_application_2.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_two", "private_key", "certs/generic_application_2.key"),
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
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "principal", "certs/generic_application_1.cer"),
					CheckBodyMatchesFile("axual_application_principal.connector_axual_application_principal_last", "private_key", "certs/generic_application_1.key"),
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

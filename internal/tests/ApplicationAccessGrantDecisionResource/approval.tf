resource "axual_application_access_grant_approval" "tf-test-decision" {
  application_access_grant = axual_application_access_grant.tf-test-grant.id
}

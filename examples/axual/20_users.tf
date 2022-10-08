resource "axual_user" "gitops_user" {
  first_name    = "Gitops_firstname"
  last_name     = "Gitops_lastname"
  email_address = "gitops_test@axual.com"
  phone_number = "+37253412559"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}
resource "axual_user" "gitops_user2" {
  first_name    = "Gitops_firstname2"
  last_name     = "Gitops_lastname2"
  email_address = "gitops_test2@axual.com"
  phone_number = "+37253412550"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}
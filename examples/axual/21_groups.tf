resource "axual_group" "gitops_test" {
  name          = "Gitops Test Group"
  phone_number="+37253412559"
  email_address="gitops.test@axual.com"
  members       = [
    axual_user.gitops_user.id,
  ]
}
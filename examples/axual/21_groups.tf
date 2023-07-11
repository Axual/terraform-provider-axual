resource "axual_group" "developers" {
  name          = "Developers"
  phone_number="+37253412559"
  email_address="gitops.test@axual.com"
  members       = [
    axual_user.jane.id,
    axual_user.john.id,
  ]
}
resource "axual_group" "team-iam" {
  name          = "testiamgroup9999"
  phone_number  = "+61123456789"
  email_address = "updated.iam@axual.com"
  members = [
    data.axual_user.test_user.id,
  ]
}

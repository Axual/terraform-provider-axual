resource "axual_group" "team-integrations" {
  name          = "testgroup9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members = [
    data.axual_user.test_user.id,
  ]
  managers = [
    data.axual_user.test_user.id,
  ]
}

data "axual_group" "team-integrations-imported" {
  name = axual_group.team-integrations.name
}
# A manager is a member with an extra role, so it needs the same person/service-account URI
# disambiguation members already need. Platform Manager rejects the whole write with 400 when the
# manager URI names the wrong collection.
resource "axual_group" "team-service-account-manager" {
  name          = "testgroupsamgr9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members = [
    data.axual_user.test_user.id,
    local.service_account_uid,
  ]
  managers = [
    data.axual_user.test_user.id,
    local.service_account_uid,
  ]
}

# A person and a service account in both lists. Each is sent as a bare uid; Platform Manager
# resolves which kind it is. A manager must also be a member, and admitting a service account
# needs Tenant Admin.
resource "axual_group" "team-service-accounts" {
  name          = "testgroupsa9999"
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

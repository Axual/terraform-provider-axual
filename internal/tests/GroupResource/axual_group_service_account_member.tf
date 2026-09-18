# A mixed member list: a person and a service account. Each needs a different collection in the
# member URI, and Platform Manager rejects the whole write with 400 when one of them is wrong.
resource "axual_group" "team-service-accounts" {
  name          = "testgroupsa9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members = [
    data.axual_user.test_user.id,
    local.service_account_uid,
  ]
}

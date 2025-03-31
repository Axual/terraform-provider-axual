resource "axual_group" "team-integrations" {
  name          = "testgroup9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members = [
    data.axual_user.test_user.id,
  ]
}

resource "axual_environment" "tf-test-env" {
  name                 = "tf-development1"
  short_name           = "tfdev"
  description          = "This is the terraform testing environment1"
  color                = "#21ccd2"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
  retention_time       = 80000
  partitions           = 1
  viewers = [
    axual_group.team-integrations.id,
  ]
}
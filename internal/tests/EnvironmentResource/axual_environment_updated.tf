resource "axual_user" "bob" {
  first_name    = "Bob"
  middle_name   = "Bar"
  last_name     = "Foo"
  email_address = "bob.foo@example.com"
  phone_number  = "+123456"
  roles = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-integrations" {
  name          = "testgroup9999"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members = [
    axual_user.bob.id,
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
  properties = {
    propertyKey1 = "propertyValue1",
    propertyKey2 = "propertyValue2"
  }
  settings = {
    enforceDataMasking = "true",
  }
  viewers = [
    axual_group.team-integrations.id,
  ]
}
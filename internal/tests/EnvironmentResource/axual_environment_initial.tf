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
  members       = [
    axual_user.bob.id,
  ]
}

resource "axual_environment" "tf-test-env" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the terraform testing environment"
  color = "#19b9be"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.team-integrations.id
  settings ={
    enforceDataMasking = "true",
  }
}
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

resource "axual_user" "bob2" {
  first_name    = "Bob2"
  middle_name   = "Bar2"
  last_name     = "Foo2"
  email_address = "bob2.foo@example.com"
  phone_number  = "+123456"
  roles = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-integrations2" {
  name          = "testgroup99990"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [
    axual_user.bob2.id,
  ]
}

resource "axual_user" "bob3" {
  first_name    = "Bob3"
  middle_name   = "Bar3"
  last_name     = "Foo3"
  email_address = "bob3.foo@example.com"
  phone_number  = "+123456"
  roles = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-integrations3" {
  name          = "testgroup99991"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [
    axual_user.bob3.id,
  ]
}

resource "axual_environment" "tf-test-env" {
  name = "tf-development1"
  short_name = "tfdev"
  description = "This is the terraform testing environment1"
  color = "#21ccd2"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.team-integrations2.id
  retention_time = 80000
  partitions = 1
  properties = {
    propertyKey1 = "propertyValue1",
    propertyKey2 = "propertyValue2"
  }
  settings = {
    enforceDataMasking = true
  }
  viewers = [
    axual_group.team-integrations2.id,
    axual_group.team-integrations3.id
  ]
}
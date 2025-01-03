resource "axual_user" "bob" {
  first_name    = "Bob"
  last_name     = "Foo"
  email_address = "bob.foo@example.com"
  phone_number = "+123456"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_user" "olivia" {
  first_name    = "Olivia"
  last_name     = "Walker"
  email_address = "olivia.walker@example.com"
  phone_number  = "+37253412554"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-group1" {
  name          = "team-group1"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.bob.id ]
}

resource "axual_group" "team-group2" {
  name          = "team-group2"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.olivia.id ]
}

resource "axual_topic" "topic-test" {
  name = "test-topic"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-group1.id
  retention_policy = "delete"
  properties = {
    propertyKey1 = "propertyValue1"
    propertyKey2 = "propertyValue2"
  }
  description = "Demo of deploying a topic via Terraform"
  viewers = [ axual_group.team-group2.id ]
}
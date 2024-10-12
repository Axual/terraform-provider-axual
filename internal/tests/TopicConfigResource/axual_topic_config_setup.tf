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

resource "axual_group" "team-group1" {
  name          = "team-group1"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.bob.id ]
}

resource "axual_environment" "tf-test-env" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "ee6e12e5301b41bf8a00ef3388806f17"
  owners = axual_group.team-group1.id
}

resource "axual_topic" "topic-test" {
  name = "test-topic"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-group1.id
  retention_policy = "delete"
  properties = {}
  description = "Demo of deploying a topic config via Terraform"
}
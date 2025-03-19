resource "axual_user" "bob" {
  first_name    = "Bob"
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
  managers = [
    axual_user.bob.id,
  ]
}
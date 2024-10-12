resource "axual_user" "bob" {
  first_name    = "Bob1"
  middle_name   = "Bar1"
  last_name     = "Foo1"
  email_address = "bob1.foo@example.com"
  phone_number  = "+1234567"
  roles = [
    { name = "APPLICATION_AUTHOR" },
    { name = "SCHEMA_AUTHOR" }
  ]
}
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

resource "axual_application" "tf-test-app" {
  name    = "tf-test-app"
  application_type     = "Custom"
  short_name = "tf_test_app_short"
  application_id = "tf.test.app"
  owners = axual_group.team-integrations.id
  type = "Java"
  visibility = "Public"
  description = "Axual's TF Test Application"
}

data "axual_application" "tf-test-app" {
  name = axual_application.tf-test-app.name
}
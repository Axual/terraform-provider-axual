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
  instance = "1be6269156d14ab09f40ea5133316a33"
  owners = axual_group.team-integrations.id
}

resource "axual_application" "tf-test-app" {
  name    = "tf-test-app"
  application_type     = "Custom"
  short_name = "tf_test_app"
  application_id = "tf.test.app"
  owners = axual_group.team-integrations.id
  type = "Java"
  visibility = "Public"
  description = "Axual's TF Test Application"
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal = "example-oauthbearer-principal"
  custom = true
}
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
  principal = file("certs/certificate.pem")
}

resource "axual_topic" "topic-test" {
  name = "test-topic"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-integrations.id
  retention_policy = "delete"
  properties = {}
  description = "Demo of deploying a topic config via Terraform"
}

data "axual_instance" "testInstance"{
  name = "testInstance"
}

resource "axual_environment" "tf-test-env" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.team-integrations.id
}

resource "axual_topic_config" "tf-topic-config" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.topic-test.id
  environment = axual_environment.tf-test-env.id
  properties = {"segment.ms"="600012", "retention.bytes"="-1"}
}

resource "axual_application_access_grant" "tf-test-application-access-grant" {
  application = axual_application.tf-test-app.id
  topic = axual_topic.topic-test.id
  environment = axual_environment.tf-test-env.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_principal.tf-test-app-principal,
    axual_topic_config.tf-topic-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-application-access-grant.id
}

data "axual_application_access_grant" "tf-test-application-access-grant" {
  application = axual_application.tf-test-app.id
  topic = axual_topic.topic-test.id
  environment = axual_environment.tf-test-env.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_access_grant.tf-test-application-access-grant
  ]
}
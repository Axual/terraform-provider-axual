resource "axual_application" "tf-test-app" {
  name             = "tf-test-app"
  application_type = "Custom"
  short_name       = "tf_test_app"
  application_id   = "tf.test.app"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Axual's TF Test Application"
}

resource "axual_environment" "tf-test-env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal = file("certs/generic-application-1.pem")
}

resource "axual_topic" "tf-test-topic" {
  name             = "test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties = {}
  description      = "Demo of deploying a topic config via Terraform"
}

resource "axual_topic_config" "tf-test-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-topic.id
  environment    = axual_environment.tf-test-env.id
  properties = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application_access_grant" "tf-test-application-access-grant" {
  application = axual_application.tf-test-app.id
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_principal.tf-test-app-principal,
    axual_topic_config.tf-test-topic-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-application-access-grant.id
}
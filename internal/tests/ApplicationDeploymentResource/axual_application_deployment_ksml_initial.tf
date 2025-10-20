resource "axual_application" "tf-test-ksml-app" {
  name              = "tf-test-ksml-app"
  application_type  = "KSML"
  short_name        = "tf_test_ksml_app"
  application_id    = "tf.test.ksml.app"
  owners            = data.axual_group.test_group.id
  visibility        = "Public"
  description       = "Axual's TF Test KSML Application"
}

resource "axual_environment" "tf-test-ksml-env" {
  name                 = "tf-ksml-development"
  short_name           = "tfksmldev"
  description          = "This is the KSML development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application_principal" "ksml_axual_application_principal" {
  environment = axual_environment.tf-test-ksml-env.id
  application = axual_application.tf-test-ksml-app.id
  principal = file("certs/connector-cert.crt")
  private_key = file("certs/connector-cert.key")
}

resource "axual_topic" "tf-test-ksml-topic" {
  name             = "ksml-test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties = {}
  description      = "Demo of KSML topic via Terraform"
}

resource "axual_topic_config" "tf-ksml-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-ksml-topic.id
  environment    = axual_environment.tf-test-ksml-env.id
  properties = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application_access_grant" "tf-test-ksml-application-access-grant" {
  application = axual_application.tf-test-ksml-app.id
  topic       = axual_topic.tf-test-ksml-topic.id
  environment = axual_environment.tf-test-ksml-env.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_principal.ksml_axual_application_principal,
    axual_topic_config.tf-ksml-topic-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-ksml-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-ksml-application-access-grant.id
}

resource "axual_application_deployment" "ksml_axual_application_deployment" {
  environment = axual_environment.tf-test-ksml-env.id
  application = axual_application.tf-test-ksml-app.id
  type = "KSML"
  definition = file("ksml-definition.yaml")
  deployment_size = "S"
  restart_policy = "on_exit"
  depends_on = [
    axual_application_access_grant_approval.tf-test-ksml-application-access-grant-approval,
  ]
}

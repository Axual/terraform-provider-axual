# AC+TLS for the export test: everything except the deployment and its state, which the test
# creates, forgets and imports back the way configuration from `terraform query` does.

resource "axual_environment" "ac_exp" {
  name                 = "tf-ds-ac-export"
  short_name           = "tfdsacexport"
  description          = "axual_application_deployment_state AC export test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "ac_exp" {
  name             = "tf-ds-ac-export-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "axual_application_deployment_state AC export test topic"
}

resource "axual_topic_config" "ac_exp" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.ac_exp.id
  environment    = axual_environment.ac_exp.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "ac_exp" {
  name              = "tf-test-app-ds-ac-export"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdsacexportapp"
  application_id    = "tf.test.ds.ac.export"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state AC export test application"
}

resource "axual_application_principal" "ac_exp" {
  environment = axual_environment.ac_exp.id
  application = axual_application.ac_exp.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active      = true
}

resource "axual_application_access_grant" "ac_exp" {
  application = axual_application.ac_exp.id
  topic       = axual_topic.ac_exp.id
  environment = axual_environment.ac_exp.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.ac_exp, axual_application_principal.ac_exp]
}

resource "axual_application_access_grant_decision" "ac_exp" {
  grant             = axual_application_access_grant.ac_exp.id
  decision          = "APPROVED"
  revoke_on_destroy = true
}

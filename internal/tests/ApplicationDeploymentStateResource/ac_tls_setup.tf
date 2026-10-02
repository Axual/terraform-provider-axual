# AC+TLS with the new resources: no target_id means legacy Axual Connect. The same flow as KC, so
# one Terraform shape works for every connector.

resource "axual_environment" "ac_tls" {
  name                 = "tf-ds-ac-tls"
  short_name           = "tfdsactls"
  description          = "axual_application_deployment_state AC+TLS test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "ac_tls" {
  name             = "tf-ds-ac-tls-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "axual_application_deployment_state AC+TLS test topic"
}

resource "axual_topic_config" "ac_tls" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.ac_tls.id
  environment    = axual_environment.ac_tls.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "ac_tls" {
  name              = "tf-test-app-ds-ac-tls"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdsactlsapp"
  application_id    = "tf.test.ds.ac.tls"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state AC+TLS test application"
}

resource "axual_application_deployment" "ac_tls" {
  application = axual_application.ac_tls.id
  environment = axual_environment.ac_tls.id
  autostart   = false
  configs = {
    "topics"          = "tf-ds-ac-tls-topic",
    "endpoint"        = "http://http-sink-target.kafka.svc.cluster.local:8080/",
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

resource "axual_application_principal" "ac_tls" {
  environment = axual_environment.ac_tls.id
  application = axual_application.ac_tls.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active      = true
  depends_on  = [axual_application_deployment.ac_tls]
}

resource "axual_application_access_grant" "ac_tls" {
  application = axual_application.ac_tls.id
  topic       = axual_topic.ac_tls.id
  environment = axual_environment.ac_tls.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.ac_tls, axual_application_principal.ac_tls]
}

resource "axual_application_access_grant_approval" "ac_tls" {
  application_access_grant = axual_application_access_grant.ac_tls.id
}

# KC+SASL: the credential can only be created once the deployment and its target exist, so the
# deployment comes first with autostart = false.

resource "axual_environment" "kc_sasl" {
  name                 = "tf-ds-kc-sasl"
  short_name           = "tfdskcsasl"
  description          = "axual_application_deployment_state KC+SASL test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "kc_sasl" {
  name             = "tf-ds-kc-sasl-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "axual_application_deployment_state KC+SASL test topic"
}

resource "axual_topic_config" "kc_sasl" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.kc_sasl.id
  environment    = axual_environment.kc_sasl.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "kc_sasl" {
  name              = "tf-test-app-ds-kc-sasl"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdskcsaslapp"
  application_id    = "tf.test.ds.kc.sasl"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state KC+SASL test application"
}

data "axual_connect_cluster" "kc_sasl" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
  id          = local.connect_cluster_sasl_id
}

resource "axual_application_deployment" "kc_sasl" {
  application    = axual_application.kc_sasl.id
  environment    = axual_environment.kc_sasl.id
  target_id      = data.axual_connect_cluster.kc_sasl.id
  target_version = "1.0.0"
  autostart      = false
  configs = {
    "topics"          = "tf-ds-kc-sasl-topic",
    "endpoint"        = "http://http-sink-target.kafka.svc.cluster.local:8080/",
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

resource "axual_application_credential" "kc_sasl" {
  environment = axual_environment.kc_sasl.id
  application = axual_application.kc_sasl.id
  target      = "KAFKA"
  depends_on  = [axual_application_deployment.kc_sasl]
}

resource "axual_application_access_grant" "kc_sasl" {
  application = axual_application.kc_sasl.id
  topic       = axual_topic.kc_sasl.id
  environment = axual_environment.kc_sasl.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.kc_sasl, axual_application_credential.kc_sasl]
}

resource "axual_application_access_grant_approval" "kc_sasl" {
  application_access_grant = axual_application_access_grant.kc_sasl.id
}

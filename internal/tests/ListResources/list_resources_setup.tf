resource "axual_environment" "tf-query-env" {
  name                 = "tf-query-env"
  short_name           = "tfqueryenv"
  description          = "Environment for the terraform query tests"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "tf-query-topic" {
  name             = "tf-query-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Topic for the terraform query tests"
}

resource "axual_topic_config" "tf-query-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-query-topic.id
  environment    = axual_environment.tf-query-env.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "tf-query-connector" {
  name              = "tf-query-connector"
  application_type  = "Connector"
  application_class = "org.apache.kafka.connect.axual.utils.LogSourceConnector"
  type              = "SOURCE"
  short_name        = "tfqueryconnector"
  application_id    = "tf.query.connector"
  owners            = data.axual_group.test_group.id
  visibility        = "Public"
  description       = "Connector application for the terraform query tests"
}

resource "axual_application_principal" "tf-query-principal" {
  environment = axual_environment.tf-query-env.id
  application = axual_application.tf-query-connector.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active      = true
}

resource "axual_application_access_grant" "tf-query-producer-grant" {
  application = axual_application.tf-query-connector.id
  topic       = axual_topic.tf-query-topic.id
  environment = axual_environment.tf-query-env.id
  access_type = "PRODUCER"
  depends_on  = [axual_topic_config.tf-query-topic-config, axual_application_principal.tf-query-principal]
}

resource "axual_application_access_grant_approval" "tf-query-producer-approval" {
  application_access_grant = axual_application_access_grant.tf-query-producer-grant.id
}

resource "axual_application_deployment" "tf-query-deployment" {
  environment = axual_environment.tf-query-env.id
  application = axual_application.tf-query-connector.id
  configs = {
    "logger.name"          = "tfquerylogger",
    "throughput"           = "10",
    "topic"                = "tf-query-topic",
    "key.converter"        = "StringConverter",
    "value.converter"      = "StringConverter",
    "config.action.reload" = "restart",
    "tasks.max"            = "1",
    "errors.tolerance"     = "none"
  }
  depends_on = [axual_application_access_grant_approval.tf-query-producer-approval]
}

resource "axual_application" "tf-query-custom" {
  name             = "tf-query-custom"
  application_type = "Custom"
  short_name       = "tfquerycustom"
  application_id   = "tf.query.custom"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  description      = "Custom application for the terraform query tests"
}

resource "axual_application_credential" "tf-query-credential" {
  environment = axual_environment.tf-query-env.id
  application = axual_application.tf-query-custom.id
  target      = "KAFKA"
}

resource "axual_application_access_grant" "tf-query-consumer-grant" {
  application = axual_application.tf-query-custom.id
  topic       = axual_topic.tf-query-topic.id
  environment = axual_environment.tf-query-env.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.tf-query-topic-config, axual_application_credential.tf-query-credential]
}

resource "axual_application_access_grant_rejection" "tf-query-consumer-rejection" {
  application_access_grant = axual_application_access_grant.tf-query-consumer-grant.id
  reason                   = "Rejected by the terraform query tests"
}

resource "axual_schema_version" "tf-query-schema" {
  body        = file("avro-schemas/tf_query_person.avsc")
  version     = "1.0.0"
  description = "Schema for the terraform query tests"
  owners      = data.axual_group.test_group.id
}

data "axual_user" "ben" {
  email = "ben.foo@example.com"
}

resource "axual_topic_browse_permissions" "tf-query-browse" {
  topic_config = axual_topic_config.tf-query-topic-config.id
  users        = [data.axual_user.ben.id]
}

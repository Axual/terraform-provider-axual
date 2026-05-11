resource "axual_topic" "tf-test-topic" {
  name             = "test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Demo of deploying a topic config via Terraform"
}

resource "axual_topic_config" "tf-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-topic.id
  environment    = axual_environment.tf-test-env.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application_access_grant" "tf-test-application-access-grant" {
  application = axual_application.tf-test-app.id
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  access_type = "PRODUCER"
  depends_on  = [axual_topic_config.tf-topic-config]
}

resource "axual_application_access_grant_approval" "tf-test-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-application-access-grant.id
}

resource "axual_application_deployment" "connector_axual_application_deployment" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  configs = {
    "logger.name"                 = "tflogger",
    "throughput"                  = "10",
    "topic"                       = "test-topic",
    "key.converter"               = "StringConverter",
    "value.converter"             = "StringConverter",
    "header.converter"            = "",
    "config.action.reload"        = "restart",
    "tasks.max"                   = "1",
    "errors.log.include.messages" = "false",
    "errors.log.enable"           = "false",
    "errors.retry.timeout"        = "0",
    "errors.retry.delay.max.ms"   = "60000",
    "errors.tolerance"            = "none",
    "predicates"                  = "",
    "topic.creation.groups"       = "",
    "transforms"                  = ""
  }
  depends_on = [axual_application_access_grant_approval.tf-test-application-access-grant-approval]
}

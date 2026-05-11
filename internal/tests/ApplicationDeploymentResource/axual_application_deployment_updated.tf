resource "axual_application_principal" "connector_axual_application_principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
}

resource "axual_application_access_grant" "tf-test-application-access-grant" {
  application = axual_application.tf-test-app.id
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  access_type = "PRODUCER"
  depends_on = [
    axual_application_principal.connector_axual_application_principal,
    axual_topic_config.tf-topic-config
  ]
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
    "tasks.max"                   = "2",
    "errors.log.include.messages" = "false",
    "errors.log.enable"           = "false",
    "errors.retry.timeout"        = "0",
    "errors.retry.delay.max.ms"   = "60000",
    "errors.tolerance"            = "none",
    "predicates"                  = "",
    "topic.creation.groups"       = "",
    "transforms"                  = ""
  }
  depends_on = [
    axual_application_access_grant_approval.tf-test-application-access-grant-approval,
  ]
}

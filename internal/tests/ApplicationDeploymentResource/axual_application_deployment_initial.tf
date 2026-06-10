resource "axual_application_principal" "connector_axual_application_principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active = true
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

# Wait for activation propagation in the platform's search index before the deployment pre-flight check.
resource "time_sleep" "wait_for_principal_activation" {
  depends_on      = [axual_application_principal.connector_axual_application_principal]
  create_duration = "3s"
  triggers = {
    principal_pem = axual_application_principal.connector_axual_application_principal.principal
  }
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
    "config.action.reload"        = "restart",
    "tasks.max"                   = "1",
    "errors.log.include.messages" = "false",
    "errors.log.enable"           = "false",
    "errors.retry.timeout"        = "0",
    "errors.retry.delay.max.ms"   = "60000",
    "errors.tolerance"            = "none",
  }
  depends_on = [
    axual_application_access_grant_approval.tf-test-application-access-grant-approval,
    time_sleep.wait_for_principal_activation,
  ]
}

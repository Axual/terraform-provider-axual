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
  depends_on = [
    axual_application_access_grant_approval.tf-test-application-access-grant-approval,
  ]
}

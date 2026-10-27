resource "axual_application_access_grant" "tf-test-grant" {
  application   = axual_application.tf-test-app.id
  topic         = axual_topic.tf-test-topic.id
  environment   = axual_environment.tf-test-env.id
  access_type   = "CONSUMER"
  request_again = "again-1"
  depends_on    = [axual_application_principal.tf-test-app-principal, axual_topic_config.tf-test-topic-config]
}

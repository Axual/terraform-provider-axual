resource "axual_application_access_grant_decision" "tf-test-decision" {
  application = axual_application.tf-test-app.id
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  access_type = "CONSUMER"
  decision    = "APPROVED"
  depends_on  = [axual_application_access_grant.tf-test-grant]
}

resource "axual_application_access_grant_decision" "tf-test-decision" {
  application = axual_application.tf-test-app.name
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  access_type = "CONSUMER"
  decision    = "DENIED"
  reason      = "Not allowed, see INC-1234"
  depends_on  = [axual_topic_config.tf-test-topic-config]
}

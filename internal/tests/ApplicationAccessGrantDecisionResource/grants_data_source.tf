data "axual_application_access_grants" "tf-test-grants" {
  topic       = axual_topic.tf-test-topic.id
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  statuses    = ["Revoked"]
}

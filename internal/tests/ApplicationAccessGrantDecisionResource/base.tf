resource "axual_application" "tf-test-app" {
  name             = "tf-test-decision-app"
  application_type = "Custom"
  short_name       = "tf_test_decision_app"
  application_id   = "tf.test.decision.app"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Axual's TF test application for grant decisions"
}

resource "axual_environment" "tf-test-env" {
  name                 = "tf-decision"
  short_name           = "tfdecision"
  description          = "Stream owner environment for grant decision tests"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = "tf-test-decision-principal"
  custom      = true
}

resource "axual_topic" "tf-test-topic" {
  name             = "tf-test-decision-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Topic for grant decision tests"
}

resource "axual_topic_config" "tf-test-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-topic.id
  environment    = axual_environment.tf-test-env.id
}

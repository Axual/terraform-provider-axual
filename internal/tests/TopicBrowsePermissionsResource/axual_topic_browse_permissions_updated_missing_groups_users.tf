resource "axual_environment" "tf-test-env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "tf-test-topic" {
  name             = "test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties = {}
  description      = "Demo of deploying a topic config via Terraform"
}

resource "axual_topic_config" "tf-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-topic.id
  environment    = axual_environment.tf-test-env.id
  properties = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_topic_browse_permissions" "tf-test-topic-browse-permissions" {
  topic_config = axual_topic_config.tf-topic-config.id
}
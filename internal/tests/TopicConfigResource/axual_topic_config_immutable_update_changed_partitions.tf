resource "axual_topic_config" "tf-topic-config-immutable" {
  partitions     = 2
  retention_time = 864000
  topic          = axual_topic.tf-test-topic-1.id
  environment    = axual_environment.tf-test-env.id
}

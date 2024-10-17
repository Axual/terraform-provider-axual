resource "axual_topic_config" "tf-topic-config" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.topic-test.id
  environment = axual_environment.tf-test-env.id
  properties = {"segment.ms"="600012", "retention.bytes"="-1"}
}
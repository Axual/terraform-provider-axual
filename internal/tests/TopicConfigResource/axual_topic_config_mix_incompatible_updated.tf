resource "axual_topic_config" "tf_test_topic_config" {
  partitions           = 2
  retention_time       = 864001
  topic                = axual_topic.tf_test_topic.id
  environment          = axual_environment.tf_test_env.id
  key_schema_version   = axual_schema_version.protobuf_v2.id
  value_schema_version = axual_schema_version.json_v3.id
  properties           = { "segment.ms" = "600013", "retention.bytes" = "2" }
  force = true
}
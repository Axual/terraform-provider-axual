resource "axual_topic" "tf-test-topic" {
  name             = "test-topic"
  key_type         = "AVRO"
  key_schema       = axual_schema_version.test_key_v1.schema_id
  value_type       = "AVRO"
  value_schema     = axual_schema_version.test_value_v1.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  description      = "Demo of deploying a topic via Terraform"
  properties       = {}
}

resource "axual_topic_config" "example-with-schema-version" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.tf-test-topic.id
  environment          = axual_environment.tf-test-env.id
  key_schema_version   = axual_schema_version.test_key_v1.id
  value_schema_version = axual_schema_version.test_value_v1.id
}

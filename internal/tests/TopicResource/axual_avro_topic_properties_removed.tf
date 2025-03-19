resource "axual_schema_version" "test_v3" {
  body = file("avro-schemas/gitops_test_v3.avsc")
  version     = "3.0.0"
  description = "Gitops test schema version 3"
}

resource "axual_topic" "topic-avro-test" {
  name             = "test-avro-topic"
  key_type         = "AVRO"
  key_schema       = axual_schema_version.test_v3.schema_id
  value_type       = "AVRO"
  value_schema     = axual_schema_version.test_v3.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  description      = "Changed Demo of deploying a topic via Terraform"
}
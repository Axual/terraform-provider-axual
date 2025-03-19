resource "axual_schema_version" "test_v1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_v2" {
  body = file("avro-schemas/gitops_test_v2.avsc")
  version     = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_topic" "topic-avro-test" {
  name             = "test-avro-topic"
  key_type         = "AVRO"
  key_schema       = axual_schema_version.test_v1.schema_id
  value_type       = "AVRO"
  value_schema     = axual_schema_version.test_v2.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties = {
    propertyKey1 = "propertyValue1"
    propertyKey2 = "propertyValue2"
  }
  description = "Demo of deploying a topic via Terraform"
}

data "axual_topic" "topic-test-imported" {
  name = axual_topic.topic-avro-test.name
}
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

resource "axual_schema_version" "protobuf_v1" {
  body = file("protobuf-schemas/tf-protobuf-test1.proto")
  version     = "1.0.0"
  description = "AddressBook schema"
  type = "PROTOBUF"
}

resource "axual_schema_version" "jsonschema_v1" {
  body = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Person schema"
  type = "JSON_SCHEMA"
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

resource "axual_topic" "topic_mix_test" {
  name             = "test-mix-topic"
  key_type         = "JSON_SCHEMA"
  key_schema       = axual_schema_version.jsonschema_v1.schema_id
  value_type       = "PROTOBUF"
  value_schema     = axual_schema_version.protobuf_v1.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "compact,delete"
  properties = {
    propertyKey1 = "propertyValue3"
    propertyKey2 = "propertyValue4"
  }
  description = "Demo of deploying a mixed schema topic via Terraform"
}

data "axual_topic" "topic_mix_test_imported" {
  name = axual_topic.topic_mix_test.name
}
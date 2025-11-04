resource "axual_environment" "tf_test_env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_schema_version" "protobuf_v1" {
  body        = file("protobuf-schemas/tf-protobuf-test1.proto")
  version     = "1.0.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}

resource "axual_schema_version" "json_v1" {
  body        = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

resource "axual_topic" "tf_test_topic" {
  name             = "test-topic"
  key_type         = "PROTOBUF"
  key_schema       = axual_schema_version.protobuf_v1.schema_id
  value_type       = "JSON_SCHEMA"
  value_schema     = axual_schema_version.json_v1.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  description      = "Demo of deploying a topic via Terraform"
  properties       = {}
}

resource "axual_schema_version" "protobuf_v2" {
  body        = file("protobuf-schemas/tf-protobuf-test2.proto")
  version     = "1.1.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}

resource "axual_schema_version" "json_v2" {
  body        = file("json-schemas/tf-json-schema-test2.json")
  version     = "1.1.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

resource "axual_schema_version" "json_v3" {
  body        = file("json-schemas/tf-json-schema-test3.json")
  version     = "2.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

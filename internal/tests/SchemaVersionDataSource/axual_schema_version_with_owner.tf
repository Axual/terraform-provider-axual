resource "axual_schema_version" "test_v1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
  owners      = data.axual_group.test_group.id
}

data "axual_schema_version" "test_v1_imported" {
  full_name = "io.axual.qa.general.GitOpsTest1"
  version   = axual_schema_version.test_v1.version
}

resource "axual_schema_version" "protobuf_v1" {
  body = file("protobuf-schemas/tf-protobuf-test1.proto")
  version     = "1.0.0"
  description = "AddressBook schema"
  owners      = data.axual_group.test_group.id
  type = "PROTOBUF"
}

data "axual_schema_version" "protobuf_v1_imported" {
  full_name = "AddressBook"
  version   = axual_schema_version.protobuf_v1.version
}

resource "axual_schema_version" "jsonschema_v1" {
  body = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Person schema"
  owners      = data.axual_group.test_group.id
  type = "JSON_SCHEMA"
}

data "axual_schema_version" "jsonschema_v1_imported" {
  full_name = "Person"
  version   = axual_schema_version.jsonschema_v1.version
}
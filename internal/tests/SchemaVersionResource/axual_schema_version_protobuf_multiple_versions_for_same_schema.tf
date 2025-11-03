resource "axual_schema_version" "test_protobuf_v1" {
  body        = file("protobuf-schemas/tf-protobuf-test1.proto")
  version     = "1.0.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}

resource "axual_schema_version" "test_protobuf_v2" {
  body        = file("protobuf-schemas/tf-protobuf-test2.proto")
  version     = "2.0.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}

resource "axual_schema_version" "test_protobuf_v3" {
  body        = file("protobuf-schemas/tf-protobuf-test3.proto")
  version     = "3.0.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}

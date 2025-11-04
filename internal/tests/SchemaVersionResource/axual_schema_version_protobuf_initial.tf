resource "axual_schema_version" "test_protobuf_v1" {
  body        = file("protobuf-schemas/tf-protobuf-test1.proto")
  version     = "1.0.0"
  description = "AddressBook schema"
  type        = "PROTOBUF"
}
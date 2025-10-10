resource "axual_schema_version" "test_v1" {
  body        = file("avro-schemas/gitops_test_1_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_v2" {
  body        = file("avro-schemas/gitops_test_1_v2_backwards_compatible.avsc")
  version     = "2.0.0"
  description = "Gitops test schema version"
}

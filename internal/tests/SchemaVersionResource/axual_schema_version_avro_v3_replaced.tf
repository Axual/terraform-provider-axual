resource "axual_schema_version" "test_v1" {
  body = file("avro-schemas/gitops_test_v3_forwards_compatible.avsc")
  version     = "3.0.0"
  description = "Gitops test schema version"
}
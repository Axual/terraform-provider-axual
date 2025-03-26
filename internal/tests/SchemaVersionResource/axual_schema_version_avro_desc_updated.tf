resource "axual_schema_version" "test_v1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version1"
}
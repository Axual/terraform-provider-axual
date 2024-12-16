resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v3_forwards_compatible.avsc")
  version = "3.0.0"
  description = "Gitops test schema version"
}
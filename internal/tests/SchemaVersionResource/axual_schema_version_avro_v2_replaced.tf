resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v2_backwards_compatible.avsc")
  version = "2.0.0"
  description = "Gitops test schema version"
}
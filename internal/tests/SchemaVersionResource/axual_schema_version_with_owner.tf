resource "axual_schema_version" "test_v2_with_owner" {
  body = file("avro-schemas/gitops_test_v2.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
  owners      = data.axual_group.test_group.id
}

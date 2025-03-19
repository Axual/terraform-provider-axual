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
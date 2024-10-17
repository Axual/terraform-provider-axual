resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

data "axual_schema_version" "axual_gitops_test_schema_version1" {
  full_name = "io.axual.qa.general.GitOpsTest1"
  version = axual_schema_version.axual_gitops_test_schema_version1.version
}
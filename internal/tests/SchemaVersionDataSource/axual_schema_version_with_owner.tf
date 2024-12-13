resource "axual_schema_version" "axual_gitops_test_schema_version_with_owner" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
  owners      = data.axual_group.user_group.id
}

data "axual_schema_version" "axual_gitops_test_schema_version_with_owner" {
  full_name = "io.axual.qa.general.GitOpsTest1"
  version   = axual_schema_version.axual_gitops_test_schema_version_with_owner.version
}
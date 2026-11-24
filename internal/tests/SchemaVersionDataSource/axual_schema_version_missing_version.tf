
data "axual_schema_version" "test_v1_missing_version" {
  full_name  = "io.axual.qa.general.GitOpsTest1"
  version    = "9.9.9"
  depends_on = [axual_schema_version.test_v1]
}

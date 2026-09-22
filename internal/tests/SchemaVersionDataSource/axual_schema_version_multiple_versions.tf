# A schema with two versions. Checks the data source can still find the older one after a
# newer one exists - see the "Schema version matching the name you requested was not found" bug.
resource "axual_schema_version" "multi_v1" {
  body        = file("avro-schemas/gitops_test_multi_version_v1.avsc")
  version     = "1.0.0"
  description = "Multi-version gitops test schema, version 1"
}

resource "axual_schema_version" "multi_v2" {
  body        = file("avro-schemas/gitops_test_multi_version_v2.avsc")
  version     = "2.0.0"
  description = "Multi-version gitops test schema, version 2"
  depends_on  = [axual_schema_version.multi_v1]
}

data "axual_schema_version" "multi_v1_imported" {
  full_name  = "io.axual.qa.general.GitOpsTestMultiVersion"
  version    = "1.0.0"
  depends_on = [axual_schema_version.multi_v2]
}

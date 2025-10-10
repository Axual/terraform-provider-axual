resource "axual_schema_version" "test_avro_explicit_type_v1" {
  body        = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version with explicit AVRO type"
  type        = "AVRO"
}

resource "axual_schema_version" "test_json_v1" {
  body        = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Gitops test JSON Schema version"
  type        = "JSON_SCHEMA"
}

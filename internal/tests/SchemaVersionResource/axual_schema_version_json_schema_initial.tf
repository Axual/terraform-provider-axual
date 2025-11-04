resource "axual_schema_version" "test_json_v1" {
  body        = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

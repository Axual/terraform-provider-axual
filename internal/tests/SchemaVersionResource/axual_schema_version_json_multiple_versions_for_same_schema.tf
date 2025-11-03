resource "axual_schema_version" "test_json_v1" {
  body        = file("json-schemas/tf-json-schema-test1.json")
  version     = "1.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}


resource "axual_schema_version" "test_json_v2" {
  body        = file("json-schemas/tf-json-schema-test2.json")
  version     = "2.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

resource "axual_schema_version" "test_json_v3" {
  body        = file("json-schemas/tf-json-schema-test3.json")
  version     = "3.0.0"
  description = "Person schema"
  type        = "JSON_SCHEMA"
}

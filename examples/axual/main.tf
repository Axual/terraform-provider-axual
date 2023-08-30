resource "axual_schema_version" "application" {
  schema = file("avro-schemas/application.avsc")
  version = "1.0.0"
  description = "Application Schema"
}
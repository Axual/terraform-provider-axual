resource "axual_environment" "tf-test-env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_schema_version" "test_key_v1" {
  body        = file("avro-schemas/avro-schema1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_key_v2" {
  body        = file("avro-schemas/avro-schema1-v2.avsc")
  version     = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_key_v3" {
  body        = file("avro-schemas/avro-schema1-v3.avsc")
  version     = "3.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_value_v1" {
  body        = file("avro-schemas/avro-schema2.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "test_value_v2" {
  body        = file("avro-schemas/avro-schema2-v2.avsc")
  version     = "2.0.0"
  description = "Gitops test schema version"
}
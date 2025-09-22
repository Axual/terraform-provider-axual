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


resource "axual_topic" "tf-test-topic" {
  name             = "test-topic"
  key_type         = "AVRO"
  key_schema       = axual_schema_version.test_key_v1.schema_id
  value_type       = "AVRO"
  value_schema     = axual_schema_version.test_value_v1.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  description      = "Demo of deploying a topic via Terraform"
  properties       = {}
}

resource "axual_topic_config" "example-with-schema-version" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.tf-test-topic.id
  environment          = axual_environment.tf-test-env.id
  key_schema_version   = axual_schema_version.test_key_v1.id
  value_schema_version = axual_schema_version.test_value_v1.id
  properties           = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

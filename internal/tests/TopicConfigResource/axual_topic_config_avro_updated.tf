resource "axual_user" "bob" {
  first_name    = "Bob"
  last_name     = "Foo"
  email_address = "bob.foo@example.com"
  phone_number = "+123456"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-group1" {
  name          = "team-group1"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.bob.id ]
}

resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/avro-schema1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version1_v2" {
  body = file("avro-schemas/avro-schema1-v2.avsc")
  version = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version2" {
  body = file("avro-schemas/avro-schema2.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version2_v2" {
  body = file("avro-schemas/avro-schema2-v2.avsc")
  version = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_environment" "tf-test-env" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "ee6e12e5301b41bf8a00ef3388806f17"
  owners = axual_group.team-group1.id
}

resource "axual_topic" "topic-test" {
  name = "test-topic"
  key_type = "AVRO"
  key_schema = axual_schema_version.axual_gitops_test_schema_version1.schema_id
  value_type = "AVRO"
  value_schema = axual_schema_version.axual_gitops_test_schema_version2.schema_id
  owners = axual_group.team-group1.id
  retention_policy = "delete"
  description = "Demo of deploying a topic via Terraform"
  properties = {}
}

resource "axual_topic_config" "example-with-schema-version" {
  partitions = 1
  retention_time = 864001
  topic = axual_topic.topic-test.id
  environment = axual_environment.tf-test-env.id
  key_schema_version = axual_schema_version.axual_gitops_test_schema_version1_v2.id
  value_schema_version = axual_schema_version.axual_gitops_test_schema_version2_v2.id
  properties = {"segment.ms"="600013", "retention.bytes"="2"}
}
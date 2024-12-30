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

resource "axual_user" "olivia" {
  first_name    = "Olivia"
  last_name     = "Walker"
  email_address = "olivia.walker@example.com"
  phone_number  = "+37253412554"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_group" "team-group1" {
  name          = "team-group1"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.bob.id ]
}

resource "axual_group" "team-group2" {
  name          = "team-group2"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [ axual_user.olivia.id ]
}

resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version2" {
  body = file("avro-schemas/gitops_test_v2.avsc")
  version = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version3" {
  body = file("avro-schemas/gitops_test_v3.avsc")
  version = "3.0.0"
  description = "Gitops test schema version 3"
}

resource "axual_topic" "topic-test" {
  name = "test-topic"
  key_type = "AVRO"
  key_schema = axual_schema_version.axual_gitops_test_schema_version3.schema_id
  value_type = "AVRO"
  value_schema = axual_schema_version.axual_gitops_test_schema_version3.schema_id
  owners = axual_group.team-group2.id
  retention_policy = "delete"
  description = "Changed Demo of deploying a topic via Terraform"
  viewers = [ axual_group.team-group1.id ]
}
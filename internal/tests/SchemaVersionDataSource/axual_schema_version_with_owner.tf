data "axual_group" "group" {
  name = "Admins"
}

# if the schemaRolesEnforced property of the tenant is set to true, we can define a new group

# resource "axual_user" "bob" {
#   first_name    = "Bob"
#   middle_name   = "Bar"
#   last_name     = "Foo"
#   email_address = "bob.foo@example.com"
#   phone_number  = "+123456"
#   roles = [
#     { name = "STREAM_AUTHOR" },
#     { name = "STREAM_ADMIN" },
#   ]
# }
#
# resource "axual_group" "team-integrations" {
#   name          = "testgroup9999"
#   phone_number  = "+6112356789"
#   email_address = "test.user@axual.com"
#   members       = [
#     axual_user.bob.id,
#   ]
# }

resource "axual_schema_version" "axual_gitops_test_schema_version_with_owner" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "Gitops test schema version"
  owners      = data.axual_group.group.id
}

data "axual_schema_version" "axual_gitops_test_schema_version_with_owner" {
  full_name = "io.axual.qa.general.GitOpsTest1"
  version   = axual_schema_version.axual_gitops_test_schema_version_with_owner.version
}
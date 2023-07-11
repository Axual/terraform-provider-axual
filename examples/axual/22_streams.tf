resource "axual_stream" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.developers.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

resource "axual_stream" "support" {
  name = "support"
  key_type = "String"
  value_type = "String"
  owners = axual_group.developers.id
  retention_policy = "delete"
  properties = { }
  description = "Support tickets from Help Desk"
}

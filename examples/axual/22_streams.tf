resource "axual_stream" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.developers.id
  retention_policy = "delete"
  properties = { }
}

resource "axual_stream" "support" {
  name = "support"
  key_type = "String"
  value_type = "String"
  owners = axual_group.developers.id
  retention_policy = "delete"
  properties = { }
}


output "logs_id" {
  description = "Logs Stream Id"
  value = axual_stream.logs.id
}

output "logs_name" {
  description = "Logs Stream Id"
    value = axual_stream.logs.name
}
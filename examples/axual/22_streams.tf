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


output "logs_id" {
  description = "Logs Stream Id"
  value = axual_stream.logs.id
}

output "logs_name" {
  description = "Logs Stream Name"
    value = axual_stream.logs.name
}

output "support_id" {
  description = "Support Stream Id"
  value = axual_stream.logs.id
}

output "support_name" {
  description = "Support Stream Name"
    value = axual_stream.logs.name
}
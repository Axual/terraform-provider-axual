data "axual_environment" "danny_env" {
  id = "d4faeb4353a04a77a0e2536bf3e654dc"
}
output "danny_env" {
  value = data.axual_environment.danny_env
}

data "axual_group" "frontend_developers" {
 id = "952497db42754bfc97cef4109f59bd2a"
}
output "frontend_developers" {
  value = data.axual_group.frontend_developers
}

data "axual_topic" "danny_topic" {
 id = "434f4763d0594b31af788c187bb63b0f"
}
output "danny_topic" {
  value = data.axual_topic.danny_topic
}

data "axual_application" "danny_app" {
 id = "e4c45499c726477d88ae8ea57c8fe230"
}
output "danny_app" {
  value = data.axual_application.danny_app
}

data "axual_schema_version" "application_v1_0_3" {
 id = "85acd22f4e3b483c9e4cbd405cc8098f"
}
output "application_v1_0_3" {
  value = data.axual_schema_version.application_v1_0_3
}

data "axual_application_access_grant" "testconnect_consume_juliusbank" {
 id = "cc56541c7e7449f99d06432431456c83"
}
output "testconnect_consume_juliusbank" {
  value = data.axual_application_access_grant.testconnect_consume_juliusbank
}

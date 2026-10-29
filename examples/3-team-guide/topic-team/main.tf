resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version2" {
  body = file("avro-schemas/gitops_test_v2.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

data "axual_group" "topic-author-team" {
  name = "Topic Author Team"
}

resource "axual_topic" "logs" {
  name = "logs"
  key_type = "AVRO"
  key_schema = axual_schema_version.axual_gitops_test_schema_version1.schema_id
  value_type = "AVRO"
  value_schema = axual_schema_version.axual_gitops_test_schema_version2.schema_id
  owners = data.axual_group.topic-author-team.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

data "axual_environment" "environment-author-team" {
  name = "tf-development"
}

resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.logs.id
  environment = data.axual_environment.environment-author-team.id
  key_schema_version = axual_schema_version.axual_gitops_test_schema_version1.id
  value_schema_version = axual_schema_version.axual_gitops_test_schema_version2.id
}

# MAIN FLOW SETUP

# The topic team's decisions on the logs topic in tf-development. The decision finds the
# application team's request itself; the two repositories can apply in either order.
locals {
  logs_dev_consumers = {
    "LogScraper" = { decision = "APPROVED" }
  }
}

resource "axual_application_access_grant_decision" "logs_dev_consumers" {
  for_each    = local.logs_dev_consumers
  application = each.key
  topic       = axual_topic.logs.id
  environment = data.axual_environment.environment-author-team.id
  access_type = "CONSUMER"
  decision    = each.value.decision
  reason      = try(each.value.reason, null)
  # The topic config goes after the decisions on destroy: a removed decision revokes its grant first,
  # and Platform Manager doesn't delete a topic config while grants are active.
  depends_on = [axual_topic_config.logs_in_dev]
}


# ALTERNATIVE FLOW SETUP
# Instead of the main flow for the same grant: the topic team creates the grant and the decision.
# Remove "LogScraper" from local.logs_dev_consumers above, and the application team doesn't create
# the grant. Otherwise two decisions and two grant resources manage the same platform grant.

# data "axual_application" "log_scraper" {
#   name = "LogScraper"
# }
#
# resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
#   application = data.axual_application.log_scraper.id
#   topic = axual_topic.logs.id
#   environment = data.axual_environment.environment-author-team.id
#   access_type = "CONSUMER"
# }
#
# resource "axual_application_access_grant_decision" "dash_consume_from_logs_in_dev" {
#   grant_id = axual_application_access_grant.dash_consume_from_logs_in_dev.id
#   decision = "APPROVED"
# }

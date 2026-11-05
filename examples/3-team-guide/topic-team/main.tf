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

# The topic team's decision on LogScraper's request to read logs in tf-development. The decision finds
# the application team's request itself; once the topic exists, the request and the decision can be
# created in either order. For many applications on one topic, see "Many applications on one topic"
# in the axual_application_access_grant_decision docs.
resource "axual_application_access_grant_decision" "log_scraper_reads_logs" {
  application = "LogScraper" # name, or UID
  topic       = axual_topic.logs.id
  environment = data.axual_environment.environment-author-team.id
  access_type = "CONSUMER"
  decision    = "APPROVED" # or "DENIED" with a reason
  # The topic config goes after the decision on destroy: a removed decision revokes its grant first,
  # and Platform Manager doesn't delete a topic config while grants are active.
  depends_on = [axual_topic_config.logs_in_dev]
}


# ALTERNATIVE FLOW SETUP
# Instead of the main flow for the same grant: the topic team creates the grant and the decision.
# Remove the main-flow decision "log_scraper_reads_logs" above, and the application team doesn't
# create the grant. Otherwise two decisions and two grant resources manage the same platform grant.
# If the instance disables self-approval, creating the decision fails (the topic team made the request
# itself) until another topic owner approves the request in the UI; then apply again.

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
#   grant_id   = axual_application_access_grant.dash_consume_from_logs_in_dev.id
#   decision   = "APPROVED"
#   depends_on = [axual_topic_config.logs_in_dev]
# }

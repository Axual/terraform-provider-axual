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

data "axual_application" "log_scraper" {
  name = "LogScraper"
}

# MAIN FLOW SETUP

data "axual_application_access_grant" "logs_producer_produce_to_logs_in_dev" {
  application = data.axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = data.axual_environment.environment-author-team.id
  access_type = "PRODUCER"
}

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval"{
  application_access_grant = data.axual_application_access_grant.logs_producer_produce_to_logs_in_dev.id
}


# ALTERNATIVE FLOW SETUP

# resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
#   application = data.axual_application.log_scraper.id
#   topic = axual_topic.logs.id
#   environment = data.axual_environment.environment-author-team.id
#   access_type = "CONSUMER"
#   # depends_on = [
#   #   axual_application_principal.log_scraper_in_dev_principal,
#   # ]
# }
#
# resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval"{
#   application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
# }
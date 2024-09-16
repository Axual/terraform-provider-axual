#
# Axual TERRAFORM PROVIDER EXAMPLE
#
# This TerraForm file shows the basic capabilities of the TerraForm provider for Axual
#

resource "axual_user" "tenant_admin" {
  first_name    = "Tenant"
  last_name     = "Admin"
  email_address = "kubernetes@axual.com"
  roles         = [
    { name = "TENANT_ADMIN" },
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" },
    { name = "STREAM_ADMIN" },
    { name = "APPLICATION_ADMIN" }
  ]
}

resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   axual_user.tenant_admin.id,
   axual_user.tenant_admin.id,
 ]
}

resource "axual_environment" "development" {
  name = "development"
  short_name = "dev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.tenant_admin_group.id
}

resource "axual_application" "log_scraper" {
  name    = "LogScraper"
  application_type     = "Custom"
  short_name = "log_scraper"
  application_id = "io.axual.gitops.scraper"
  owners = axual_group.tenant_admin_group.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for finding all Logs for developers"
}

resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_topic" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.tenant_admin_group.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.log_scraper_in_dev_principal ]
}

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval"{
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
  depends_on = [axual_topic_config.logs_in_dev]
}
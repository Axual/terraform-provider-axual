#Logged in Terraform User(by default kubernetes@axual.com) needs to have application admin rights(for create access request) and stream admin rights(for revoking access request) or be owner of the application and the stream (by being user in the same group as the application's and stream's owner group)
resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.dev_dashboard.id
  stream = axual_stream.logs.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_dev_principal ]
}

resource "axual_application_access_grant" "log_scraper_consume_from_support_in_dev" {
  application = axual_application.log_scraper.id
  stream = axual_stream.support.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.log_scraper_in_dev_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_staging" {
  application = axual_application.dev_dashboard.id
  stream = axual_stream.logs.id
  environment = axual_environment.staging.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_staging_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_support_in_staging" {
  application = axual_application.dev_dashboard.id
  stream = axual_stream.support.id
  environment = axual_environment.staging.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_staging_principal ]
}

resource "axual_application_access_grant" "scraper_produce_to_logs_in_staging" {
  application = axual_application.log_scraper.id
  stream = axual_stream.logs.id
  environment = axual_environment.staging.id
  access_type = "PRODUCER"
  depends_on = [ axual_application_principal.log_scraper_in_staging_principal ]
}



output "dash_consume_from_logs_in_staging_id" {
  description = "Id of Access grant for Dev Dashboard to consume from Logs in Staging"
  value = axual_application_access_grant.dash_consume_from_logs_in_staging.id
}

output "dash_consume_from_logs_in_staging_status" {
  description = "Status of Access grant for Dev Dashboard to consume from Logs in Staging"
  value = axual_application_access_grant.dash_consume_from_logs_in_staging.status
}






resource "axual_application_access_grant" "dash_consume_from_logs_in_production" {
  application = axual_application.dev_dashboard.id
  stream = axual_stream.logs.id
  environment = axual_environment.production.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_production_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_support_in_production" {
  application = axual_application.dev_dashboard.id
  stream = axual_stream.support.id
  environment = axual_environment.production.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_production_principal ]
}

resource "axual_application_access_grant" "scraper_produce_to_logs_in_production" {
  application = axual_application.log_scraper.id
  stream = axual_stream.logs.id
  environment = axual_environment.production.id
  access_type = "PRODUCER"
  depends_on = [ axual_application_principal.log_scraper_in_production_principal ]
}

output "dash_consume_from_logs_in_production_id" {
  description = "Id of Access grant for Dev Dashboard to consume from Logs in Production"
  value = axual_application_access_grant.dash_consume_from_logs_in_production.id
}

output "dash_consume_from_logs_in_production_status" {
  description = "Status of Access grant for Dev Dashboard to consume from Logs in Production"
  value = axual_application_access_grant.dash_consume_from_logs_in_production.status
}
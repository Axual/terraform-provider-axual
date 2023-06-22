resource "axual_application_access_grant_authorization" "dash_consume_logs_example" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_example.id
  status = "Approved" # Change this to Revoked in order to revoke approval. That can only be done after this current state has been applied
}

output "dash_consume_logs_example_status" {
  value = axual_application_access_grant_authorization.dash_consume_logs_example.status
}

resource "axual_application_access_grant_authorization" "scraper_consume_support_example" {
  application_access_grant = axual_application_access_grant.log_scraper_consume_from_support_in_example.id
  status = "Revoked" # Auto approved can be revoked
}

output "scraper_consume_support_example_status" {
  value = axual_application_access_grant_authorization.scraper_consume_support_example.status
}


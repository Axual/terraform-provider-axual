resource "axual_application_access_grant_rejection" "scraper_produce_logs_staging_rejection" {
  application_access_grant = axual_application_access_grant.scraper_produce_to_logs_in_staging.id
}


resource "axual_application_access_grant_approval" "dash_consume_logs_dev" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}

resource "axual_application_access_grant_approval" "dash_consume_logs_staging" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_staging.id
}

resource "axual_application_access_grant_approval" "dash_consume_support_production"{
  application_access_grant = axual_application_access_grant.dash_consume_from_support_in_production.id
}

resource "axual_application_access_grant_approval" "log_consume_support_dev"{
  application_access_grant = axual_application_access_grant.log_scraper_consume_from_support_in_dev.id
}

resource "axual_application_access_grant_approval" "dash_consume_logs_production"{
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_production.id
}

resource "axual_application_access_grant_approval" "scraper_produce_logs_production"{
  application_access_grant = axual_application_access_grant.scraper_produce_to_logs_in_production.id
}

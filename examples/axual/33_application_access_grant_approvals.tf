resource "axual_application_access_grant_approval" "dash_consume_logs_dev" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}

resource "axual_application_access_grant_approval" "dash_consume_logs_staging" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_staging.id
}


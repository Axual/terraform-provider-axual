data "axual_group" "application-author-team" {
  name = "Application Author Team"
}

resource "axual_application" "log_scraper" {
  name    = "LogScraper"
  application_type     = "Custom"
  short_name = "log_scraper"
  application_id = "io.axual.gitops.scraper"
  owners = data.axual_group.application-author-team.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for finding all Logs for developers"
}

data "axual_environment" "environment-author-team" {
  name = "tf-development"
}

resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = data.axual_environment.environment-author-team.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

data "axual_topic" "topic-logs" {
  name = "logs"
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.log_scraper.id
  topic = data.axual_topic.topic-logs.id
  environment = data.axual_environment.environment-author-team.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_principal.log_scraper_in_dev_principal,
  ]
}

# The topic team's answer. A plan doesn't show an approval as a change: the new status shows after
# the next apply, or with terraform apply -refresh-only.
output "logs_access" {
  value = {
    status = axual_application_access_grant.dash_consume_from_logs_in_dev.status
    reason = axual_application_access_grant.dash_consume_from_logs_in_dev.decision_reason
  }
}

# Warns (it doesn't fail) until the topic team has approved the request (Terraform 1.5 or later).
check "logs_access_approved" {
  assert {
    condition     = axual_application_access_grant.dash_consume_from_logs_in_dev.status == "Approved"
    error_message = "Access to logs is ${axual_application_access_grant.dash_consume_from_logs_in_dev.status}."
  }
}

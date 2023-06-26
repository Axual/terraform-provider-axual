resource "axual_application" "dev_dashboard" {
  name    = "DeveloperDashboard"
  application_type     = "Custom"
  short_name = "dev_dash"
  application_id = "io.axual.devs.dashboard"
  owners = axual_group.developers.id
  type = "Java"
  visibility = "Public"
  description = "Dashboard with crucial information for Developers"
}

resource "axual_application" "log_scraper" {
  name    = "LogScraper"
  application_type     = "Custom"
  short_name = "log_scraper"
  application_id = "io.axual.gitops.scraper1"
  owners = axual_group.developers.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for finding all Logs for developers"
}

output "dashboard_id" {
  description = "Dashboard Application ID"
  value = axual_application.dev_dashboard.id
}

output "dashboard_name" {
  description = "Dashboard Application Name"
  value = axual_application.dev_dashboard.name
}

output "dashboard_short_name" {
  description = "Dashboard Application Short Name"
  value = axual_application.dev_dashboard.short_name
}
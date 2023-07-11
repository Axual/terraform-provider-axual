resource "axual_application" "dev_dashboard" {
  name    = "DeveloperDashboard"
  application_type     = "Custom"
  short_name = "dev_dash"
  application_id = "io.axual.devs.dashboard"
  owners = axual_group.developers.id
  type = "Java"
  visibility = "Public"
  description = "Dashboard with crucial information for Developers"
#  depends_on = [axual_stream_config.logs_in_production, axual_stream.support] # This is a workaround when all resources get deleted at once, to delete stream_config and stream before application. Mentioned in index.md
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
#  depends_on = [axual_stream_config.logs_in_dev, axual_stream.logs] # This is a workaround when all resources get deleted at once, to delete stream_config and stream before application. Mentioned in index.md
}

resource "axual_application_principal" "dev_dashboard_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.dev_dashboard.id
  principal = file("${path.module}/test3.pem")
}

resource "axual_application_principal" "dev_dashboard_in_staging_principal" {
  environment = axual_environment.staging.id
  application = axual_application.dev_dashboard.id
  principal = file("${path.module}/test3.pem")
}

resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.log_scraper.id
  principal = file("${path.module}/test3.pem")
}

resource "axual_application_principal" "log_scraper_in_staging_principal" {
  environment = axual_environment.staging.id
  application = axual_application.log_scraper.id
  principal = file("${path.module}/test3.pem")
}

resource "axual_application_principal" "dev_dashboard_in_production_principal" {
  environment = axual_environment.production.id
  application = axual_application.dev_dashboard.id
  principal = file("${path.module}/test3.pem")
}

resource "axual_application_principal" "log_scraper_in_production_principal" {
  environment = axual_environment.production.id
  application = axual_application.log_scraper.id
  principal = file("${path.module}/test3.pem")
}

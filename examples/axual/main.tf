# This TerraForm file shows the basic capabilities of the TerraForm provider for Axual

# The Terraform provider cannot be used to create a user. Please ensure that a user already exists before proceeding. To verify, please try logging into the UI.
# Look up yourself by e-mail – change the address
data "axual_user" "my-user" {
  email = "<your_email>"
}

# Replace with the short name of your instance
data "axual_instance" "testInstance"{
  name = "dta"
}

############################
# 1️⃣  Group
############################
resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   data.axual_user.my-user.id,
 ]
}

############################
# 2️⃣  Environment
############################
resource "axual_environment" "development" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the TF development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.tenant_admin_group.id
}

############################
# 3️⃣  Application
############################
resource "axual_application" "log_scraper" {
  name    = "tf-application"
  application_type     = "Custom"
  short_name = "tf_application"
  application_id = "io.axual.gitops.scraper"
  owners = axual_group.tenant_admin_group.id
  type = "Java"
  visibility = "Public"
  description = "TF Test Application"
}

# Principal for the log_scraper app in the ‘development’ environment
# Please make sure the certificate matches the CA of the instance
resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

# Alternatively for SASL
# Credentials for the log_scraper app in the ‘development’ environment
# resource "axual_application_credential" "creds" {
#   application = axual_application.log_scraper.id
#   environment = axual_environment.development.id
#   target      = "KAFKA"
# }
############################
# 4️⃣  Schema & Topic
############################
resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_topic" "logs" {
  name = "tf-topic"
  key_type = "String"
  value_type = "String"
  owners = axual_group.tenant_admin_group.id
  retention_policy = "delete"
  properties = { }
  description = "TF test topic"
}

resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000  # 10 days
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  properties = {"segment.ms"="600012", "retention.bytes"="-1"}
}

############################
# 5️⃣  Access & Approval
############################
resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_principal.log_scraper_in_dev_principal,
    axual_topic_config.logs_in_dev
  ]
}

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval"{
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}
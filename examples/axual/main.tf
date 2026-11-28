# This Terraform file shows the basic capabilities of the Terraform provider for Axual

# The Terraform provider cannot be used to create a user. Please ensure that a user already exists in the self-service before proceeding. To verify, please try logging into the UI.
# Look up yourself by e-mail - change the address
data "axual_user" "my-user" {
  email = "<your_email>"
}

# The uid of the service account that the provider signs in with (see provider.tf).
# It must be a member of the group that owns the resources below. Without that, it can
# create them, but it cannot create the principal, topic config, grant and decision, and
# it cannot change or delete anything. A tenant admin can add a service account to a group.
variable "service_account_uid" {
  type = string
}

# Replace with the short name of your instance
data "axual_instance" "testInstance"{
  short_name = "dta"
}

############################
# 1️⃣  Group
############################
resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   data.axual_user.my-user.id,
   var.service_account_uid,
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
# Use your own certificate: it must chain to a CA certificate of the instance.
# The certificate in certs/ only works on an instance that trusts the Axual dummy CA.
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
  # Members of the owner group may delete the schema. Without owners, a delete can need SCHEMA_ADMIN.
  owners = axual_group.tenant_admin_group.id
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
  retention_time = 864000000  # 10 days, in milliseconds
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

resource "axual_application_access_grant_decision" "connector_access" {
  grant_id = axual_application_access_grant.dash_consume_from_logs_in_dev.id
  decision = "APPROVED"
}
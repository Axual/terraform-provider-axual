# Axual Provider

Axual's Terraform Provider integrates Axual's Self-Service for Apache Kafka into Terraform, enabling users to manage Kafka configurations through infrastructure as code. Self-Service offers fine-grained access control, visibility into topic metadata, and management of topic properties, allowing users to monitor and control their Kafka streaming environment efficiently. Read more: https://docs.axual.io/axual/2024.2/self-service/index.html

## Example Usage

First, make sure to configure the connection to the Trial Account:

```terraform
terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = "2.4.0"
    }
  }
}

# Provider Configuration for local Axual platform installation

provider "axual" {
  # (String) URL that will be used by the client for all resource requests
  apiurl   = "https://platform.local/api"
  # (String) Axual realm used for the requests
  realm    = "axual"
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "kubernetes@axual.com"
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "PLEASE_CHANGE_PASSWORD"
  # (String) Client ID to be used for OAUTH
  clientid = "self-service"
  # (String) Token url
  authurl  = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  # (List of String) OAuth authorization server scopes
  scopes   = ["openid", "profile", "email"]
}
```

The following example demonstrates the basic functionality of Axual Self-Service. For more advanced features, refer to the 'Resources' and 'Guides' sections.

```terraform
#
# Axual TERRAFORM PROVIDER EXAMPLE
#
# This TerraForm file shows the basic capabilities of the TerraForm provider for Axual
#

resource "axual_user" "tenant_admin" {
  first_name    = "Tenant"
  last_name     = "Admin"
  email_address = "kubernetes@axual.com"
  roles         = [
    { name = "TENANT_ADMIN" },
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" },
    { name = "STREAM_ADMIN" },
    { name = "APPLICATION_ADMIN" }
  ]
}

resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   axual_user.tenant_admin.id,
   axual_user.tenant_admin.id,
 ]
}

resource "axual_environment" "development" {
  name = "development"
  short_name = "dev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.tenant_admin_group.id
}

resource "axual_application" "log_scraper" {
  name    = "LogScraper"
  application_type     = "Custom"
  short_name = "log_scraper"
  application_id = "io.axual.gitops.scraper"
  owners = axual_group.tenant_admin_group.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for finding all Logs for developers"
}

resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_topic" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.tenant_admin_group.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.log_scraper_in_dev_principal ]
}

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval"{
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
  depends_on = [axual_topic_config.logs_in_dev]
}
```

To create all the resources in this example, the logged-in user (defined in provider.tf) must have the following roles:

- **TENANT_ADMIN** - Required for creating the resources `axual_user` and `axual_group`
- **TOPIC_ADMIN** - for creating resources: `axual_topic`, `axual_topic_config`, `axual_application_access_grant_approval`
- **APPLICATION_ADMIN** - for creating resources: `axual_application`, `axual_application_principal`, `axual_application_access_grant`
- **ENVIRONMENT_ADMIN** - for creating resource: `axual_environment`


# Getting started
## Required User Roles
- The Terraform User who is logged in (With a Trial account, the default username kubernetes@axual.com), needs to have at least both of the following user roles:
  - **APPLICATION_ADMIN** - for creating `axual_application`, `axual_application_principal`, `axual_application`
  - **STREAM_ADMIN** - for revoking access request
- Alternatively, they can be the owner of both the application and the topic, which entails being a user in the same group as the owner group of the application and topic.


## Compatibility
| Terraform Provider Version | Supported Platform Manager Version(s) |
|----------------------------|---------------------------------------|
| 2.1.0                      | 7.0.7 - 8.4.x                        |
| 2.2.0                      | 8.5.x                                |
| 2.3.0                      | 8.5.x                                |
| 2.4.0                      | 8.6.0+                               |

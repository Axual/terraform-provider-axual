# Axual Provider

The Axual Terraform Provider integrates Axual's Self-Service for Apache Kafka with Terraform, making it easy to manage Axual's Kafka configurations as code. It offers detailed access control, clear topic visibility, and simple topic settings management, enabling users to monitor and control their Kafka streaming setup effectively.

Learn more about Axual Self-Service: https://docs.axual.io/axual/2024.4/self-service/index.html

## Example Usage
- There are two authentication modes supported by the provider:  **Auth0** and **Keycloak**.

### Auth0 Authentication

- Please use this provider configuration if the authentication is against Auth0. Auth0 is used by the Axual Trial environment:

```hcl
provider "axual" {
  # Default `authMode` is "keycloak", if omitted.
  authmode = "auth0"
  # (String) URL that will be used by the client for all resource requests
  apiurl   = "https://app.axual.cloud/api"
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "PLEASE_CHANGE_USERNAME"
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "PLEASE_CHANGE_PASSWORD"
  # (String) Client ID to be used for OAUTH
  clientid = "eY6aEMAO8XAkoKE9e9pZFcOs7Wxs6VBQ"
  # (String) Token url
  authurl  = "https://axual.eu.auth0.com/oauth/token"
  # (List of String) OAuth authorization server scopes
  scopes   = ["openid", "profile", "email"]
  # The audience for OAuth. Usually the same as `apiurl`.
  audience = "https://app.axual.cloud/api/"
}
```

### Keycloak Authentication

- Please use this provider configuration if the authentication is against **Keycloak**. Keycloak is used in `axual cloud` and when deploying Axual Streaming and Axual Governance on Kubernetes:

```hcl
provider "axual" {
  # Default `authMode` is "keycloak", if omitted.
  authmode = "keycloak"
  # URL that will be used by the client for all resource requests
  apiurl   = "https://axual.cloud/api"
  # Axual realm used for the requests
  realm    = "PLEASE_CHANGE_TENANT_NAME"
  # Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "PLEASE_CHANGE_USERNAME"
  # (Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "PLEASE_CHANGE_PASSWORD"
  # Client ID to be used for OAUTH
  clientid = "self-service"
  # Token url
  authurl  = "https://axual.cloud/auth/realms/PLEASE_CHANGE_TENANT_NAME/protocol/openid-connect/token"
  # OAuth authorization server scopes
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
# - When trying out this example:
# - replace `instance` name from `Dev Test Acceptance` to the name of your instance.
# - Please use `terraform import axual_user.tenant_admin <USER UID FROM UI>` so the user matches the user you are logged in as


resource "axual_user" "tenant_admin" { # Please use `terraform import axual_user.tenant_admin <USER UID FROM UI>`
  first_name    = "Tenant"
  last_name     = "Admin"
  email_address = "kubernetes@axual.com"
  roles         = [
    { name = "TENANT_ADMIN" },
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" },
    { name = "SCHEMA_AUTHOR" },
  ]
}

resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   axual_user.tenant_admin.id,
 ]
}

data "axual_instance" "testInstance"{
  name = "Dev Test Acceptance" # Replace with the name of your instance
}

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
  retention_time = 864000
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  properties = {"segment.ms"="600012", "retention.bytes"="-1"}
}

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
```

To create all the resources in this example, the logged-in user (defined in provider.tf and imported using `terraform import`) must have the following roles:

- **TENANT_ADMIN** - Required for creating the resources `axual_user` and `axual_group`
- **SCHEMA_AUTHOR** - for creating resource: `axual_schema_version`
- **STREAM_AUTHOR** - for creating resources: `axual_topic`, `axual_topic_config`, `axual_application_access_grant_approval`
- **APPLICATION_AUTHOR** - for creating resources: `axual_application`, `axual_application_principal`, `axual_application_access_grant`
- **ENVIRONMENT_AUTHOR** - for creating resource: `axual_environment`

## Distributed Gitops multi-repo example
- The Axual Terraform provider supports distinct team roles:
  - The Admin Team manages environments, users and groups.
  - The Topic Team handles topic creation, configuration, and access approvals.
  - The Application Team manages applications, deployments, and requests permission to produce to or consume from a topic owned by the Topic Team.
- These capabilities enable a GitOps workflow where teams manage their Terraform states independently and collaborate through resource references using Terraform data sources and approvals.
- The Guide: [Multi-Repo Guide](guides/multi-repo.md)
  - Please follow the guide for a setup where 3 teams have separated responsibilities.

**Key Practices**
- Each team manages their own Terraform state independently.
- Teams utilize dedicated Terraform users configured with the minimum required privileges.
- Teams reference resources from other teams by utilizing Terraform data sources.

## Compatibility
| Terraform Provider Version | Supported Platform Manager Version(s) |
|----------------------------|---------------------------------------|
| 2.1.x                      | 7.0.7 - 8.4.x                        |
| 2.2.x                      | 8.5.x                                |
| 2.3.x                      | 8.5.x                                |
| 2.4.x                      | 8.6.x                                |
| 2.5.x                      | 9.1.x                                |

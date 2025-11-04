# Axual Provider
The **Axual Terraform Provider** integrates Axual's Self-Service for Apache Kafka with Terraform, making it easy to manage Kafka configurations as code. It provides:

- Fine-grained access control
- Controlling access to and visibility of topics using Kafka ACLs
- Simplified configuration management

This empowers teams to monitor, manage, and automate their Kafka streaming environments efficiently.

Learn more: [Axual Self-Service Documentation](https://docs.axual.io/axual/2025.3/self-service/index.html)

## Example Usage

### Step 1 – Initialize the provider

Create a file named `provider.tf` and paste the following block. Be sure to replace the placeholder values with your actual credentials and configuration.

```hcl
terraform {
  required_providers {
    axual = {
      source  = "axual/axual"
      version = "2.7.1"
    }
  }
}

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

### Step 2 – Define Resources

Before using the provider:

- Ensure users are created in both Axual Self-Service and the authentication system.
- Use the `axual_user` data source to reference yourself.
- Use the `axual_instance` data source to reference existing clusters.

#### Full Example
- The following example demonstrates the basic functionality of Axual Self-Service. For more advanced features, refer to the 'Resources' and 'Guides' sections.

```terraform
# This TerraForm file shows the basic capabilities of the TerraForm provider for Axual

# The Terraform provider cannot be used to create a user. Please ensure that a user already exists before proceeding. To verify, please try logging into the UI.
# Look up yourself by e-mail – change the address
data "axual_user" "my-user" {
  email = "<your_email>"
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
```


Once set up, run:

```
terraform apply
```

This will:

- Create your group and development environment
- Register the application and principal
- Deploy a topic and ACL
- Approve application access


#### Required Roles

To run the above configuration, your user must have the following roles:

| Role                 | Required For                                                                         |
|----------------------|--------------------------------------------------------------------------------------|
| `TENANT_ADMIN`       | `axual_user`, `axual_group`                                                          |
| `SCHEMA_AUTHOR`      | `axual_schema_version`                                                               |
| `STREAM_AUTHOR`      | `axual_topic`, `axual_topic_config`, `axual_application_access_grant_approval`       |
| `APPLICATION_AUTHOR` | `axual_application`, `axual_application_principal`, `axual_application_access_grant` |
| `ENVIRONMENT_AUTHOR` | `axual_environment`                                                                  |

### Step 3 – Verify & Continue
- Go to `/overview` in Axual Self-Service UI to confirm the application is producing to the topic.
- Connect any Kafka client (e.g. Java) using the created certificate or credentials.
- Terraform will store sensitive values (such as credentials) in the `terraform.tfstate` file — please ensure that it is properly secured.

## GitOps: Multi-Repo Architecture

The Axual Terraform provider enables a distributed GitOps setup across teams:

| Team         | Responsibilities                                   |
|--------------|----------------------------------------------------|
| Admin Team   | Manages users, groups, environments                |
| Topic Team   | Owns topics and defines access policies            |
| App Team     | Deploys applications and requests topic access     |

[See Guide: Multi-Repo GitOps Setup](guides/multi-repo.md)

**Best Practices:**

- Each team manages its own Terraform state
- Teams use minimum-permission service users
- Resources are shared via data sources

##  Limitations
- Terraform-created users are **not** added to the authentication system (e.g. Keycloak), and cannot log in.
  - Instead, use `data.axual_user` to reference existing users
  - Or import them: [`terraform import`](https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/user#import)

## Compatibility
| Terraform Provider Version | Supported Platform Manager Versions  |
|----------------------------|--------------------------------------|
| 2.1.x                      | 7.0.7 - 8.4.x                        |
| 2.2.x                      | 8.5.x                                |
| 2.3.x                      | 8.5.x                                |
| 2.4.x                      | 8.6.x – 9.0.x                        |
| 2.5.x                      | 9.1.x - onward                       |
| 2.6.x                      | 12.0.x - onward                      |
| 2.7.x                      | 12.0.x - onward                      |

## Custom JSON Schema Support

Enable IDE integration for Terraform auto-complete and validation by importing the provider’s custom JSON schema:

[Custom JSON Schema Guide](guides/json-schema)
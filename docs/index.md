# Axual Provider
The **Axual Terraform Provider** integrates Axual's Self-Service for Apache Kafka with Terraform, making it easy to manage Kafka configurations as code. It provides:

- Fine-grained access control
- Controlling access to and visibility of topics using Kafka ACLs
- Simplified configuration management

This empowers teams to monitor, manage, and automate their Kafka streaming environments efficiently.

Learn more: [Axual Self-Service Documentation](https://docs.axual.io/axual/2026.3/self-service/index.html)

## Example Usage

### Step 1 – Initialize the provider

Create a file named `provider.tf` and paste the following block. Be sure to replace the placeholder values with your actual credentials and configuration.

```hcl
terraform {
  required_providers {
    axual = {
      source  = "axual/axual"
      version = ">= 3.3.0" # the first release with axual_application_access_grant_decision
    }
  }
}

variable "axual_client_secret" {
  description = "Client secret of the service account. Set it with TF_VAR_axual_client_secret. To use AXUAL_CLIENT_SECRET instead, remove this variable and the client_secret line below."
  type        = string
  sensitive   = true
}

provider "axual" {
  # URL that will be used by the client for all resource requests
  apiurl        = "https://axual.cloud/api"
  # Axual realm used for the requests. This is your tenant's short name.
  realm         = "PLEASE_CHANGE_TENANT_NAME"
  # Token url
  authurl       = "https://axual.cloud/auth/realms/PLEASE_CHANGE_TENANT_NAME/protocol/openid-connect/token"

  # Client ID of the service account. It can be omitted if the environment variable AXUAL_CLIENT_ID is set.
  client_id     = "PLEASE_CHANGE_SERVICE_ACCOUNT_CLIENT_ID"
  # (Sensitive) Client secret of the service account, shown once when the account is created
  # or its secret is rotated. It can be omitted if the environment variable AXUAL_CLIENT_SECRET is set.
  client_secret = var.axual_client_secret
}
```

A tenant admin creates the service account and gives you its client ID and secret. The
secret is shown once, so keep it somewhere your pipeline can read it and pass it in with
`AXUAL_CLIENT_SECRET` rather than writing it into a `.tf` file. The tenant admin also gives the
service account the roles in "Required Roles" below, and adds it to the group that
owns your resources.

Service accounts need Platform Manager 16.0.0 or later. On Platform Manager 15.0.x, use the
deprecated `username` and `password` instead (see the guide below).

Service accounts can also authenticate with **no secret at all**, using a token from your
own identity provider. See the
[Service account authentication](guides/service-account-authentication) guide for that,
and for how to move an existing configuration off a username and password.

~> **Username and password authentication is deprecated.** Earlier versions authenticated
as a person using the OAuth2 password grant. That still works and emits a warning, but it
puts a real person's password into automation and will be removed in a future release.

### Step 2 – Define Resources

Before using the provider:

- Ensure users are created in both Axual Self-Service and the authentication system.
- Use the `axual_user` data source to reference yourself.
- Set `service_account_uid` to the uid of the service account from Step 1. The example adds it to
  the group that owns all resources. Without that, the service account can create the resources,
  but it cannot create the principal, topic config, grant and decision, and it cannot change or
  delete anything. On Platform Manager 15.0.x (username and password), remove `service_account_uid`
  from the example.
- Use the `axual_instance` data source to reference an existing instance.

#### Full Example
- The following example demonstrates the basic functionality of Axual Self-Service. For more advanced features, refer to the 'Resources' and 'Guides' sections.

```terraform
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

The same identity requests and approves the access. This works only when self-approval is on for
the instance (the default).

#### Required Roles

To run the above configuration, the identity in the provider block (normally the service account)
must have the following roles. A role lets it create a resource. To change or delete the resource,
and to create a principal, topic config, grant or decision, it must also be a member of the group
in `owners`. The example does this with `service_account_uid`.

| Role                 | Required For                                                                         |
|----------------------|--------------------------------------------------------------------------------------|
| `TENANT_ADMIN`       | `axual_group`                                                                        |
| `SCHEMA_AUTHOR`      | `axual_schema_version`                                                               |
| `STREAM_AUTHOR`      | `axual_topic`, `axual_topic_config`, `axual_application_access_grant_decision`       |
| `APPLICATION_AUTHOR` | `axual_application`, `axual_application_principal`, `axual_application_access_grant` |
| `ENVIRONMENT_AUTHOR` | `axual_environment`                                                                  |

### Step 3 – Verify & Continue
- Go to the topic in the Axual Self-Service UI to confirm the application has consume access.
- Connect any Kafka client (e.g. Java) using the created certificate or credentials.
- Terraform will store sensitive values (such as credentials) in the `terraform.tfstate` file - please ensure that it is properly secured.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `apiurl` (String) URL that will be used by the client for all resource requests
- `authurl` (String) Token URL

### Optional

- `audience` (String, Deprecated) Audience for OAUTH.
- `authmode` (String, Deprecated) Authentication mode.
- `client_id` (String) Client ID of the service account. Not used by federated service accounts, which Keycloak identifies from the assertion. Can be omitted if the environment variable `AXUAL_CLIENT_ID` is set.
- `client_secret` (String, Sensitive) Client secret of the service account. Setting it selects service account authentication with a secret. Can be omitted if the environment variable `AXUAL_CLIENT_SECRET` is set. Needs Platform Manager 16.0.0 or later.
- `clientid` (String, Deprecated) Client ID to be used for oauth.
- `oidc_token` (String, Sensitive) The assertion a federated service account authenticates with: a token issued by your own identity provider, not an Axual token. Setting it selects federated authentication, which sends no client secret at all. Can be omitted if the environment variable `AXUAL_OIDC_TOKEN` is set, which is the usual way to supply it from a CI pipeline. Needs Platform Manager 16.0.0 or later.
- `oidc_token_file` (String) Path to a file holding the assertion a federated service account authenticates with. The file is read again every time the access token is renewed, so use this rather than `oidc_token` when something rewrites the assertion as it rotates. Can be omitted if the environment variable `AXUAL_OIDC_TOKEN_FILE` is set. Needs Platform Manager 16.0.0 or later.
- `password` (String, Sensitive) Password belonging to the user. Can be omitted if the environment variable `AXUAL_AUTH_PASSWORD` is set. Deprecated: use a service account instead.
- `realm` (String) Axual realm used for the requests. This is your tenant's short name.
- `scopes` (List of String) OAuth authorization server scopes. Only applies to username and password authentication; service account authentication does not send a scope.
- `username` (String) Username for all requests. Will be used to acquire a token. Can be omitted if the environment variable `AXUAL_AUTH_USERNAME` is set. Deprecated: use a service account instead.

## GitOps: Multi-Repo Architecture

The Axual Terraform provider enables a distributed GitOps setup across teams:

| Team         | Responsibilities                                         |
|--------------|----------------------------------------------------------|
| Admin Team   | Manages users, groups, environments, instances, clusters |
| Topic Team   | Owns topics and defines access policies                  |
| App Team     | Deploys applications and requests topic access           |

[See Guide: Multi-Repo GitOps Setup](guides/multi-repo.md)

**Best Practices:**

- Each team manages its own Terraform state
- Teams use minimum-permission service users
- Resources are shared via data sources

##  Limitations
- Terraform cannot create users. Create them in the authentication system (e.g. Keycloak) and in Self-Service first.
  - Then use `data.axual_user` to reference existing users
  - Or import them: [`terraform import`](https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/user#import)

## Compatibility
| Terraform Provider Version | Supported Platform Manager Versions  | Supported Platform Versions |
|----------------------------|--------------------------------------|-----------------------------|
| 2.1.x                      | 7.0.7 - 8.4.x                        |   2023.3 - 2024.1           |
| 2.2.x                      | 8.5.x                                |   2024.1                    |
| 2.3.x                      | 8.5.x                                |   2024.1                    |
| 2.4.x                      | 8.6.x – 9.0.x                        |   2024.2 - 2024.4           |
| 2.5.0 - 2.5.6              | 9.1.x - 11.0.3                       |   2024.4 - 2025.2           |
| 2.5.7                      | 11.0.4 - 13.0.2                      |   2025.2 - 2025.4           |
| 2.6.x                      | 12.0.x - 13.0.2                      |   2025.3 - 2025.4           |
| 2.7.x                      | 12.0.x - 13.0.2                      |   2025.3 - 2025.4           |
| 2.8.x                      | 12.0.x - 13.0.2                      |   2025.3 - 2025.4           |
| 2.9.x                      | 12.0.x - 13.0.2                      |   2025.3 - 2025.4           |
| 3.0.x                      | 12.0.x - 14.0.4                      |   2025.3 - 2026.1           |
| 3.1.x                      | 15.0.x - onward                      |   2026.2 - onwards          |
| 3.2.x                      | 15.0.x - onward                      |   2026.2 - onwards          |
| 3.3.x                      | 15.0.x - onward                      |   2026.2 - onwards          |

Some resources and attributes need a newer Platform Manager than 15.0.x. Service accounts,
`axual_connect_cluster` and Connector deployments on a Kafka Connect cluster need 16.0.0 or later.
`axual_flink_cluster` needs 17.0.0 or later with 3.3.x, and 3.2.x cannot manage an
`axual_flink_cluster` on Platform Manager 17.0.0 or later. Each page says which version it needs.

## IDE support (IntelliJ)

Get syntax highlighting, validation and auto-completion of Axual resources in IntelliJ:

[IDE support (IntelliJ) guide](guides/json-schema)
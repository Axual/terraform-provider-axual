# Terraform Provider for Axual Platform

The Terraform Provider for Axual Platform allows you to manage Axual resources through Terraform, including applications, topics, schemas, environments, users, and groups.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- Access to an Axual Platform instance (local deployment or Axual Cloud)

## Using the Provider

### Installation

The provider is available on the [Terraform Registry](https://registry.terraform.io/). Add it to your Terraform configuration:

```terraform
terraform {
  required_providers {
    axual = {
      source  = "axual/axual"
      version = "~> 2.6"
    }
  }
}
```

### Provider Configuration

#### Local Axual Platform

Configure the provider to connect to a locally deployed Axual Platform instance:

```terraform
provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "axual"
  username = "kubernetes@axual.com"   # or set using env: AXUAL_AUTH_USERNAME
  password = "PLEASE_CHANGE_PASSWORD" # or set using env: AXUAL_AUTH_PASSWORD
  clientid = "self-service"
  authurl  = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
}
```

#### Axual Cloud

Configure the provider to connect to Axual Cloud:

```terraform
provider "axual" {
  apiurl   = "https://axual.cloud/api"
  realm    = "YOUR_REALM_NAME"
  username = "YOUR_USERNAME"
  password = "YOUR_PASSWORD"
  clientid = "self-service"
  authurl  = "https://axual.cloud/auth/realms/YOUR_REALM_NAME/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
}
```

### Authentication

The provider supports authentication via:
- Direct credentials in the provider block
- Environment variables: `AXUAL_AUTH_USERNAME` and `AXUAL_AUTH_PASSWORD`

### Example Usage

Here's a complete example that creates a group, environment, application, topic, and access grant:

```terraform
terraform {
  required_providers {
    axual = {
      source  = "axual/axual"
      version = "~> 2.6"
    }
  }
}

provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "axual"
  username = "kubernetes@axual.com"
  password = "PLEASE_CHANGE_PASSWORD"
  clientid = "self-service"
  authurl  = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  scopes   = ["openid", "profile", "email"]
}

# Look up existing user
data "axual_user" "my-user" {
  email = "your.email@example.com"
}

# Look up existing instance
data "axual_instance" "my-instance" {
  short_name = "dev"
}

# Create a group
resource "axual_group" "developers" {
  name    = "Development Team"
  members = [
    data.axual_user.my-user.id,
  ]
}

# Create an environment
resource "axual_environment" "development" {
  name                   = "Development"
  short_name             = "dev"
  description            = "Development environment"
  color                  = "#19b9be"
  visibility             = "Public"
  authorization_issuer   = "Stream owner"
  instance               = data.axual_instance.my-instance.id
  owners                 = axual_group.developers.id
}

# Create an application
resource "axual_application" "my-app" {
  name             = "My Application"
  application_type = "Custom"
  short_name       = "my_app"
  application_id   = "com.example.myapp"
  owners           = axual_group.developers.id
  type             = "Java"
  visibility       = "Public"
  description      = "My streaming application"
}

# Create application principal for SSL authentication
resource "axual_application_principal" "my-app-principal" {
  environment = axual_environment.development.id
  application = axual_application.my-app.id
  principal   = file("path/to/certificate.pem")
}

# Create a topic
resource "axual_topic" "events" {
  name             = "my-events"
  key_type         = "String"
  value_type       = "String"
  owners           = axual_group.developers.id
  retention_policy = "delete"
  description      = "Application events topic"
}

# Configure topic in environment
resource "axual_topic_config" "events-dev" {
  partitions      = 3
  retention_time  = 864000  # 10 days
  topic           = axual_topic.events.id
  environment     = axual_environment.development.id
  properties      = {
    "segment.ms" = "600000"
  }
}

# Grant application access to topic
resource "axual_application_access_grant" "my-app-consume-events" {
  application = axual_application.my-app.id
  topic       = axual_topic.events.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on  = [
    axual_application_principal.my-app-principal,
    axual_topic_config.events-dev
  ]
}

# Approve the access grant
resource "axual_application_access_grant_approval" "approve-access" {
  application_access_grant = axual_application_access_grant.my-app-consume-events.id
}
```

After creating your configuration, initialize and apply:

```bash
terraform init
terraform plan
terraform apply
```

For more examples, see the [`examples/`](./examples/) directory in this repository.

## Documentation

Full provider documentation, including all available resources and data sources, is available on the [Terraform Registry](https://registry.terraform.io/).

## Contributing

Contributions are welcome! If you'd like to contribute to the development of this provider, please see [DEVELOPERS.md](DEVELOPERS.md) for information on:
- Setting up a local development environment
- Building the provider from source
- Running tests
- Debugging

## Support

For issues, questions, or feature requests, please open an issue in this repository.

## License

See [LICENSE](LICENSE) for details.
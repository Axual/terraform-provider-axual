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

After configuring the provider, initialize and apply your Terraform configuration:

```bash
terraform init
terraform plan
terraform apply
```

For more examples, see the `examples/` directory in this repository.

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
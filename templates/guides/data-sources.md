---
page_title: "Using Data Sources"
---

Data sources allow you to reference existing resources in Axual Platform Manager without managing them through Terraform.

## When to Use Data Sources

Use data sources when:

- **Multi-repository setups**: Reference resources managed by another team's Terraform configuration
- **Existing infrastructure**: Reference resources that were created manually or through other tools
- **Read-only access**: Reference resources you can read but not modify
- **Cross-team collaboration**: Reference shared resources like groups, environments, or schemas owned by other teams

Data sources are read-only - they fetch information but don't create, update, or delete resources.

## Available Data Sources

| Data Source | Lookup By | Attributes Returned |
|-------------|-----------|---------------------|
| `axual_user` | `email` | `id`, `first_name`, `last_name`, `middle_name`, `phone_number` |
| `axual_group` | `name` | `id`, `members`, `email_address`, `phone_number` |
| `axual_instance` | `name` or `short_name` | `id`, `name`, `short_name`, `description` |
| `axual_environment` | `name` or `short_name` | `id`, `name`, `short_name`, `description`, `color`, `visibility`, `authorization_issuer`, `owners`, `instance`, `retention_time`, `partitions`, `properties` |
| `axual_topic` | `name` | `id`, `description`, `key_type`, `key_schema`, `value_type`, `value_schema`, `owners`, `retention_policy`, `properties` |
| `axual_application` | `name` or `short_name` | `id`, `name`, `short_name`, `description`, `application_type`, `application_id`, `application_class`, `type`, `owners`, `visibility` |
| `axual_schema_version` | `full_name` and `version` | `id`, `schema_id`, `body`, `description`, `owners` |
| `axual_application_access_grant` | `application`, `topic`, `environment`, `access_type` | `id`, `status` |

## Example Usage

### axual_user

To define an `axual_user` data source, provide the user's email:

```hcl
data "axual_user" "tom" {
  email = "tom@email.com"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_group" "team-integrations" {
  name          = "Integrations group"
  phone_number  = "+123456"
  email_address = "integrationsgroup@axual.com"
  members       = [
    data.axual_user.tom.id
  ]
}
```

**Available attributes:** `id`, `email`, `first_name`, `last_name`, `middle_name`, `phone_number`

### axual_group

To define an `axual_group` data source, provide the group name:

```hcl
data "axual_group" "frontend_developers" {
  name = "Frontend Developers"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_topic" "logs" {
  name             = "logs"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.frontend_developers.id
  retention_policy = "delete"
  properties       = {}
  description      = "Dev topic of type string"
}
```

**Available attributes:** `id`, `name`, `members`, `email_address`, `phone_number`

### axual_instance

To define an `axual_instance` data source, provide either the instance `name` or `short_name`:

```hcl
data "axual_instance" "test_instance" {
  short_name = "test"
}

# Alternative: lookup by name
data "axual_instance" "test_instance_by_name" {
  name = "Test Instance"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_environment" "test" {
  name                 = "test"
  short_name           = "test"
  description          = "This is the development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = axual_group.tenant_admin_group.id
}
```

**Available attributes:** `id`, `name`, `short_name`, `description`

### axual_environment

To define an `axual_environment` data source, provide either the environment `name` or `short_name`:

```hcl
data "axual_environment" "dev" {
  name = "development"
}

# Alternative: lookup by short_name
data "axual_environment" "dev_by_short_name" {
  short_name = "dev"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_topic_config" "logs_in_dev" {
  partitions     = 1
  retention_time = 1001000
  topic          = axual_topic.logs.id
  environment    = data.axual_environment.dev.id
  properties     = { "segment.ms" = "60002", "retention.bytes" = "100" }
}
```

**Available attributes:** `id`, `name`, `short_name`, `description`, `color`, `visibility`, `authorization_issuer`, `owners`, `instance`, `retention_time`, `partitions`, `properties`

### axual_topic

To define an `axual_topic` data source, provide the topic name:

```hcl
data "axual_topic" "logs" {
  name = "logs"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_topic_config" "logs_in_dev" {
  partitions     = 1
  retention_time = 1001000
  topic          = data.axual_topic.logs.id
  environment    = data.axual_environment.dev.id
  properties     = { "segment.ms" = "60002", "retention.bytes" = "100" }
}
```

**Available attributes:** `id`, `name`, `description`, `key_type`, `key_schema`, `value_type`, `value_schema`, `owners`, `retention_policy`, `properties`

### axual_application

To define an `axual_application` data source, provide either the application `name` or `short_name`:

```hcl
data "axual_application" "logs_producer" {
  name = "logs_producer"
}

# Alternative: lookup by short_name
data "axual_application" "logs_producer_by_short_name" {
  short_name = "logs_prod"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_application_access_grant" "logs_producer_produce_to_logs_in_dev" {
  application = data.axual_application.logs_producer.id
  topic       = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "PRODUCER"
}
```

**Available attributes:** `id`, `name`, `short_name`, `description`, `application_type`, `application_id`, `application_class`, `type`, `owners`, `visibility`

### axual_schema_version

To define an `axual_schema_version` data source, provide the schema full name (`<NAMESPACE>.<NAME>`) and the version:

```hcl
data "axual_schema_version" "ApplicationV1" {
  full_name = "io.axual.qa.general.Application"
  version   = "1.0.0"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_topic" "avro_topic" {
  name             = "avro_topic"
  key_type         = "AVRO"
  key_schema       = data.axual_schema_version.ApplicationV1.schema_id
  value_type       = "AVRO"
  value_schema     = data.axual_schema_version.ApplicationV1.schema_id
  owners           = data.axual_group.frontend_developers.id
  retention_policy = "delete"
  properties       = {}
  description      = "Avro topic created using external data source"
}
```

**Available attributes:** `id`, `full_name`, `version`, `body`, `schema_id`, `description`, `owners`

### axual_application_access_grant

To define an `axual_application_access_grant` data source, provide the application id, topic id, environment id, and access type (`PRODUCER` or `CONSUMER`):

```hcl
data "axual_application_access_grant" "logs_producer_grant" {
  application = data.axual_application.logs_producer.id
  topic       = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "PRODUCER"
}
```

Now we can use this data source when creating a resource:

```hcl
resource "axual_application_access_grant_approval" "logs_producer_approval" {
  application_access_grant = data.axual_application_access_grant.logs_producer_grant.id
}
```

**Available attributes:** `id`, `application`, `topic`, `environment`, `access_type`, `status`

---
page_title: "Exporting existing resources with terraform query"
---

With `terraform query` you can turn resources that already exist in Axual Platform Manager, for
example resources created in the Self-Service UI, into Terraform configuration. Terraform finds the
resources for you, so you do not need to look up the ID of every resource.

This needs Terraform 1.14 or newer.

## When to use it

- You built resources by hand (in the UI or the API) and now want Terraform to manage them.
- You want to copy resources to another environment or another Axual installation.
- You want to split resources over the repositories of several teams, for example by owner group.

## Step 1: provider configuration

Use the normal provider configuration in a `.tf` file:

```terraform
terraform {
  required_providers {
    axual = {
      source = "axual/axual"
    }
  }
}

provider "axual" {
  apiurl        = "https://self-service.example.com/api"
  authurl       = "https://self-service.example.com/auth/realms/axual/protocol/openid-connect/token"
  realm         = "axual"
  client_id     = "YOUR_SERVICE_ACCOUNT_CLIENT_ID"     # replace with your service account
  client_secret = "YOUR_SERVICE_ACCOUNT_CLIENT_SECRET" # replace with your service account secret
}
```

## Step 2: a query file

Put one `list` block per resource type in a file whose name ends in `.tfquery.hcl`. The `config`
block holds the filters. Leave it out to list everything you can see.

```terraform
# export.tfquery.hcl
list "axual_topic" "team_a" {
  provider = axual
  config {
    owners = "Team A" # group name or group ID
  }
}

list "axual_application" "team_a" {
  provider = axual
  config {
    owners = "Team A"
  }
}

list "axual_topic_config" "team_a" {
  provider = axual
  config {
    owners      = "Team A"
    environment = "dev" # environment short name, name or ID
  }
}
```

A `list` block returns at most 100 resources unless you set `limit`, for example `limit = 1000`.

## Step 3: run the query

```shell
terraform init
terraform query
```

Terraform prints one line per resource it found:

```text
list.axual_topic.team_a         id=b032a258b08047b0bc52a23e519fef2c   orders
list.axual_topic.team_a         id=c751693eb0104ae488ee9a2b9331c9d7   payments
list.axual_application.team_a   id=6a899c1e793a43e4a6efb087aca25e94   order-service
```

## Step 4: generate the configuration

```shell
terraform query -generate-config-out=generated.tf
```

`generated.tf` now has a `resource` block and an `import` block for every resource found:

```terraform
resource "axual_topic" "team_a_0" {
  provider         = axual
  name             = "orders"
  key_type         = "String"
  value_type       = "String"
  owners           = "9f12688e64d5417ab780fd104e9e9712"
  retention_policy = "delete"
  properties       = {}
}

import {
  to       = axual_topic.team_a_0
  provider = axual
  identity = {
    id = "b032a258b08047b0bc52a23e519fef2c"
  }
}
```

## Step 5: review, then plan and apply

Rename the resources, replace IDs with references where you like (for example
`owners = data.axual_group.team_a.id`), and move the blocks into your own files. Then:

```shell
terraform plan
terraform apply
```

The plan shows the resources as `will be imported`. Nothing is created again.

## Things to know

- **Secrets are not exported.** Terraform leaves sensitive values out of generated configuration and
  writes `null # sensitive` instead. Fill them in before you apply:
  - `axual_application_deployment`: `configs`, `definition` (KSML) and `sql_script` (Flink).
    **Applying with `configs = null` clears the connector configuration.**
  - `axual_application_credential`: the password is never returned. A credential applied to another
    environment or installation is a new credential with a new password.
- **Schema versions**: Terraform writes a JSON body with `jsonencode(...)`, which changes only the
  whitespace. The first apply stores the new text and does not change the schema on the platform.
- **IDs belong to one installation.** Generated configuration refers to groups, environments and
  other resources by ID. To apply it in another environment, tenant or installation, replace the IDs
  with references or data sources first, and remove the `import` blocks.
- **Users, instances and clusters** cannot be created with Terraform. `axual_user` can be listed, but
  only to manage users that already exist.
- **Deployment states**: `axual_application_deployment_state` lists only deployments that are
  `Running` or `Stopped`. A resource that cannot be read is skipped with a warning, so one broken
  resource does not stop the whole export.
- You only see what your user or service account may see in Self-Service.

## Filters

| List resource | Filters |
|---|---|
| `axual_group` | `name` |
| `axual_user` | `email`, `role` |
| `axual_environment` | `name`, `owners` |
| `axual_application` | `name`, `short_name`, `owners`, `application_type` |
| `axual_topic` | `name`, `owners` |
| `axual_topic_config` | `topic`, `environment`, `owners` |
| `axual_topic_browse_permissions` | `topic`, `environment`, `owners` |
| `axual_schema_version` | `schema` |
| `axual_application_credential` | `application`, `environment`, `owners` |
| `axual_application_access_grant` | `application`, `topic`, `environment`, `owners`, `access_type`, `status` |
| `axual_application_access_grant_approval` | `application`, `topic`, `environment`, `owners`, `access_type` |
| `axual_application_access_grant_rejection` | `application`, `topic`, `environment`, `owners`, `access_type` |
| `axual_application_deployment` | `application`, `environment`, `owners` |
| `axual_application_deployment_state` | `application`, `environment`, `owners` |

- `name`, `short_name` and `email` match when the value contains the text, ignoring case.
- `owners`, `environment`, `application` and `topic` take an ID or an exact name (an environment or
  application also by its short name). For `axual_topic_config`, `axual_application_credential` and
  similar resources, `owners` means the owner of the topic or application they belong to.

Two resources have no list resource:

- `axual_application_principal`: a new certificate replaces the principal with a new one (a new ID)
  in the same apply, and Terraform does not allow the identity of a resource to change. Its
  certificate and private key could not be exported anyway. Import a principal by ID as before.
- `axual_flink_cluster`: it needs an instance and cluster ID, and has no list resource yet.

## Importing a single resource by identity

Every resource except `axual_flink_cluster` and `axual_application_principal` also accepts an
`identity` in an `import` block, as an alternative to `id`:

```terraform
import {
  to = axual_topic.orders
  identity = {
    id = "b032a258b08047b0bc52a23e519fef2c"
  }
}
```

`axual_application_access_grant_approval` and `axual_application_access_grant_rejection` use
`application_access_grant` as the identity attribute, and `axual_topic_browse_permissions` uses
`topic_config`, the same values they take as an import ID.

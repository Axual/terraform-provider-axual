---
page_title: "Quick start: export your resources with terraform query"
---

This page shows in five steps how to turn resources you made in the Self-Service UI into Terraform
files. You need Terraform 1.14 or newer.

For all filters and details, see [Exporting existing resources with terraform query](export-with-terraform-query).

## 1. Make a folder with the provider

Create an empty folder with a file `main.tf`:

```terraform
terraform {
  required_providers {
    axual = {
      source = "axual/axual"
    }
  }
}

provider "axual" {
  apiurl        = "YOUR_API_URL"       # for example https://self-service.example.com/api
  authurl       = "YOUR_AUTH_URL"      # the token URL of your realm
  realm         = "YOUR_REALM"         # your tenant's short name
  client_id     = "YOUR_SERVICE_ACCOUNT_CLIENT_ID"
  client_secret = "YOUR_SERVICE_ACCOUNT_CLIENT_SECRET"
}
```

## 2. Say what to export

In the same folder, create `export.tfquery.hcl`. This one finds the topics and applications of one
team:

```terraform
list "axual_topic" "team" {
  provider = axual
  config {
    owners = "YOUR_GROUP_NAME"
  }
}

list "axual_application" "team" {
  provider = axual
  config {
    owners = "YOUR_GROUP_NAME"
  }
}
```

## 3. See what is found

```shell
terraform init
terraform query
```

Terraform prints one line per resource:

```text
list.axual_topic.team         id=b032a258b08047b0bc52a23e519fef2c   orders
list.axual_application.team   id=6a899c1e793a43e4a6efb087aca25e94   order-service
```

## 4. Write the Terraform files

```shell
terraform query -generate-config-out=generated.tf
```

`generated.tf` now has a `resource` block and an `import` block for every resource found.

## 5. Let Terraform manage them

```shell
terraform plan
terraform apply
```

The plan says `will be imported` for each resource; nothing is created or changed. After the apply,
`terraform plan` says `No changes`, and the resources are managed by Terraform.

## Good to know

- Secret values (certificates, private keys, connector `configs`) are not exported. Terraform writes
  `null # sensitive`; fill them in before step 5. For a deployment, the plan stops with
  `Missing configs` until you do.
- Move `generated.tf` away before you run `terraform query` again.
- A list returns at most 100 resources; add `limit = 1000` inside the `list` block for more.

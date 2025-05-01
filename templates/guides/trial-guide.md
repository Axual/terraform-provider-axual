---
page_title: "Connecting Terraform to the Axual Trial Environment (Auth0)"
---

Getting Terraform talking to the Axual SaaS trial is a **three-step** process:

1. **Sign up for a free trial**
2. **Configure the Terraform provider with Auth0**
3. **Create resources in your personal trial cluster**

---

## Step 1 – Request a trial account

1. Go to **Get started → Start free trial** at <https://axual.com/get-started>.
2. Click **Sign up**.
3. Pick **“Try Axual Cluster”**.
4. Enter your *Organisation name* and your *first* and *last* name, then finish the signup flow.

---

## Step 2 – Initialise the provider (Auth0 flow)

Create a `provider.tf` and paste the **provider** block below.

```hcl
terraform {
  required_providers {
    axual = {
      source  = "axual/axual"
      version = "<Replace with latest version>"
    }
  }
}

provider "axual" {
  apiurl   = "https://app.axual.cloud/api"

  # 🔑 Replace these two lines
  username = "<INSERT_USERNAME>"   # your trial e-mail
  password = "<INSERT_PASSWORD>"   # your trial password

  clientid = "eY6aEMAO8XAkoKE9e9pZFcOs7Wxs6VBQ"

  authurl  = "https://axual.eu.auth0.com/oauth/token"
  scopes   = ["openid", "profile", "email"]
  audience = "https://app.axual.cloud/api"
  authmode = "auth0"
}
```


---

## Step 3 – Reference existing objects & create your own

Because the trial cluster is pre-seeded with core objects:

- Users are created only during self-registration, so use a **axual_user** data source to look yourself up.

- Clusters / Instances already exist. Reference them with **axual_instance** data source.


```hcl
#️ Look up yourself by e-mail – change the address
data "axual_user" "me" {
  email = "my-email@email.com"
}

# Pick the shared trial instance by name
data "axual_instance" "trial" {
  name = "Non-Production"
}
```
Now build your own group, environment, application, topic, ACLs, etc.

```hcl
############################
# 1️⃣  Group
############################
resource "axual_group" "my_group" {
  name    = "My Test Group"
  members = [data.axual_user.me.id]
}

############################
# 2️⃣  Environment
############################
resource "axual_environment" "dev" {
  name                  = "tf-development"
  short_name            = "tfdev"
  description           = "Terraform development environment"
  color                 = "#19b9be"
  visibility            = "Public"
  authorization_issuer  = "Stream owner"
  instance              = data.axual_instance.trial.id
  owners                = axual_group.my_group.id
}

############################
# 3️⃣  Application
############################
resource "axual_application" "app" {
  name             = "tf-application"
  application_type = "Custom"
  short_name       = "tf_application"
  application_id   = "io.axual.gitops.test"
  type             = "Java"
  visibility       = "Public"
  description      = "TF Test Application"
  owners           = axual_group.my_group.id
}

# Credentials for the app in the ‘dev’ environment
resource "axual_application_credential" "creds" {
  application = axual_application.app.id
  environment = axual_environment.dev.id
  target      = "KAFKA"
}

############################
# 4️⃣  Schema & Topic
############################
resource "axual_schema_version" "gitops_schema_v1" {
  body        = file("avro-schemas/gitops_test_v1.avsc")
  version     = "1.0.0"
  description = "GitOps test schema version"
}

resource "axual_topic" "logs" {
  name              = "logs"
  key_type          = "String"
  value_type        = "String"
  description       = "Logs from all applications"
  retention_policy  = "delete"
  properties        = {
    propertyKey1 = "propertyValue1"
    propertyKey2 = "propertyValue2"
  }
  owners            = axual_group.my_group.id
}

resource "axual_topic_config" "logs_dev" {
  topic           = axual_topic.logs.id
  environment     = axual_environment.dev.id
  partitions      = 1
  retention_time  = 864000            # 10 days
  properties      = {
    "segment.ms"      = "600012"
    "retention.bytes" = "-1"
  }
}

############################
# 5️⃣  Access & Approval
############################
resource "axual_application_access_grant" "produce_logs" {
  application  = axual_application.app.id
  topic        = axual_topic.logs.id
  environment  = axual_environment.dev.id
  access_type  = "PRODUCER"

  depends_on = [
    axual_application_credential.creds,
    axual_topic_config.logs_dev
  ]
}

resource "axual_application_access_grant_approval" "produce_logs_approval" {
  application_access_grant = axual_application_access_grant.produce_logs.id
}
```

Run **terraform apply** and confirm the plan – Terraform will:
- Create a group containing you.
- Create a development environment in the shared trial instance.
- Register an application + credentials.
- Provision a topic with ACL-s in that environment.
- Grant & approve produce rights for the application on the topic.

## What’s next?

- Open **/overview** in the Axual Self-Service UI → you should visually see the application producing to logs.
- Wire any Java (or other) client to the cluster using the credential JSON returned by the **axual_application_credential**.
  - Since password is a sensitive field, Terraform Provider will not print the password in **terraform plan** or **terraform apply** output.
  - Terraform will save the password and username in local state file(**terraform.tfstate**). Please make sure that this file is appropriately secured.
- Produce using for example a Java client and Browse messages in UI using Topic Browse to verify messages delivery.
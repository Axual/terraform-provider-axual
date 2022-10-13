---
page_title: "Terraform setup with Axual Trial environment"
---

## Axual Trial setup

- First please request a trial environment by filling in the form https://axual.com/trial/
- Please confirm your e-mail
- You will receive an e-mail with trial credentials
- Please fill in provider like this:

```shell
terraform {
  required_providers {
    axual = {
      source = "Axual/axual"
      version = ">= 1.0.0"
    }
  }
}
 provider "axual" {
   apiurl   = "https://selfservice.axual.cloud/api"
   realm    = "<REPLACE_WITH_REALM> " # Replace realm with the realm from the URL in the email you received: https://selfservice.axual.cloud/login/<REPLACE_WITH_REALM>
   username = "<REPLACE_WITH_USERNAME>"
   password = "<REPLACE_WITH_PASSWORD>"
   clientid = "self-service"
   authurl = "https://selfservice.axual.cloud/auth/realms/<REPLACE_WITH_REALM>/protocol/openid-connect/token" # Replace realm with the realm from the URL in the email you received: https://selfservice.axual.cloud/login/<REPLACE_WITH_REALM>
   scopes = ["openid", "profile", "email"]
 }
```
- Next, let's test if everything works:
```shell
terraform init
```
- When we see "**Terraform has been successfully initialized!**" the setup was successful.

- Most resources require a group or user id, so we need to import them beforehand. Please follow our guide on how to import user and group: [Importing user and group](guides/importing-user-and-groups.md)
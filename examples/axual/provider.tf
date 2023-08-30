
  terraform {
  required_providers {
    axual = {
      source  = "axual.com/hackz/axual"
    }
  }
}


# PROVIDER CONFIGURATION
#
# Below example configuration is for when you have deployed Axual Platform locally. Contact your administrator if you
# need the details for your organization's installation.
#
provider "axual" {
  apiurl   = "https://mgmt-qa.cloud.axual.io/api"
  realm    = "axual"
  username = "daniel_a" #- or set using env property export AXUAL_AUTH_USERNAME=
  password = "overreact1" #- or set using env property export AXUAL_AUTH_PASSWORD=
  clientid = "self-service"
  authurl = "https://mgmt-qa.cloud.axual.io/auth/realms/axual/protocol/openid-connect/token"
  scopes = ["openid", "profile", "email"]
}
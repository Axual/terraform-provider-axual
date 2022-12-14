terraform {
  required_providers {
    axual = {
      source = "Axual/axual"
      version = ">= 1.0.0"
    }
  }
}

provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "axual"
  username = "kubernetes@axual.com" #- or set using env property export AXUAL_AUTH_USERNAME=
  password = "PLEASE_CHANGE_PASSWORD" #- or set using env property export AXUAL_AUTH_PASSWORD=
  clientid = "self-service"
  authurl = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  scopes = ["openid", "profile", "email"]
}
terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = "2.8.1"
    }
  }
}

# Provider Configuration for local Axual platform installation

provider "axual" {
  # (String) URL that will be used by the client for all resource requests
  apiurl   = "https://platform.local/api"
  # (String) Axual realm used for the requests
  realm    = "local"
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "application_team"
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "<Change to application team password>"
  # (String) Client ID to be used for OAUTH
  clientid = "self-service"
  # (String) Token url
  authurl  = "https://platform.local/auth/realms/local/protocol/openid-connect/token"
  # (List of String) OAuth authorization server scopes
  scopes   = ["openid", "profile", "email"]
}

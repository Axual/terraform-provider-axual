terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = "3.1.0"
    }
  }
}

provider "axual" {
  # Configuration options
  # (String) URL that will be used by the client for all resource requests
  apiurl = "https://self-service.qa.np.westeurope.azure.axual.cloud/api"
  # (String) Axual realm used for the requests
  realm = "axual"
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "admin-axual"
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "notsecret"
  # (String) Client ID to be used for OAUTH
  clientid = "self-service"
  # (String) Token url
  authurl = "https://self-service.qa.np.westeurope.azure.axual.cloud/auth/realms/axual/protocol/openid-connect/token"
  # (List of String) OAuth authorization server scopes
  scopes = ["openid", "profile", "email"]
}

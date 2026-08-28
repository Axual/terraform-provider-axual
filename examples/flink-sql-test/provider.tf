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
  apiurl = ""
  # (String) Axual realm used for the requests
  realm = ""
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = ""
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = ""
  # (String) Client ID to be used for OAUTH
  clientid = ""
  # (String) Token url
  authurl = ""
  # (List of String) OAuth authorization server scopes
  scopes = ["openid", "profile", "email"]
}

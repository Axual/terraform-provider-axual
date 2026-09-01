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
  apiurl = "YOUR_API_URL"
  # (String) Axual realm used for the requests
  realm = "YOUR_REALM"
  # (String) Username for all requests. Will be used to acquire a token. It can be omitted if the environment variable AXUAL_AUTH_USERNAME is used.
  username = "YOUR_USERNAME"
  # (String, Sensitive) Password belonging to the user. It can be omitted if the environment variable AXUAL_AUTH_PASSWORD is used.
  password = "YOUR_PASSWORD"
  # (String) Client ID to be used for OAUTH
  clientid = "YOUR_CLIENT_ID"
  # (String) Token url
  authurl = "YOUR_AUTH_URL"
  # (List of String) OAuth authorization server scopes
  scopes = ["openid", "profile", "email"]
}

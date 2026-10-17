terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = ">= 3.3.0" # the first release with terraform query
    }
  }
}

provider "axual" {
  # (String) URL that will be used by the client for all resource requests
  apiurl = "YOUR_API_URL"
  # (String) Axual realm used for the requests. This is your tenant's short name.
  realm = "YOUR_REALM"
  # (String) Token url
  authurl = "YOUR_AUTH_URL"
  # (String) Client ID of the service account. It can be omitted if the environment variable AXUAL_CLIENT_ID is set.
  client_id = "YOUR_SERVICE_ACCOUNT_CLIENT_ID"
  # (String, Sensitive) Client secret of the service account. It can be omitted if the environment variable AXUAL_CLIENT_SECRET is set.
  client_secret = "YOUR_SERVICE_ACCOUNT_CLIENT_SECRET"
}

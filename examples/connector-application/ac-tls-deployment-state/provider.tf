terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = ">= 3.3.0" # the first release with axual_application_access_grant_decision
    }
  }
}

provider "axual" {
  # (String) URL that will be used by the client for all resource requests
  apiurl = "YOUR_API_URL"
  # (String) Axual realm used for the requests. This is your tenant's short name.
  realm = "YOUR_REALM"
  # (String) Token url, in the same realm
  authurl = "YOUR_AUTH_URL"
  # (String) Client ID of the service account. It can be omitted if the environment variable AXUAL_CLIENT_ID is set.
  client_id = "YOUR_SERVICE_ACCOUNT_CLIENT_ID"
  # The client secret of the service account is read from the environment variable AXUAL_CLIENT_SECRET.
  # Service accounts need Platform Manager 16.0.0 or later. On 15.0.x, use username and password instead.
}

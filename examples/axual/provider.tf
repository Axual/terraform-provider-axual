terraform {
  required_providers {
    axual = {
      source  = "Axual/axual"
      version = "3.1.0"
    }
  }
}

variable "axual_client_secret" {
  description = "Client secret of the service account. Set it with TF_VAR_axual_client_secret, or omit client_secret below and set AXUAL_CLIENT_SECRET instead."
  type        = string
  sensitive   = true
}

provider "axual" {
  # Configuration options
  # (String) URL that will be used by the client for all resource requests
  apiurl        = "https://platform.local/api"
  # (String) Axual realm used for the requests. This is your tenant's short name.
  realm         = "axual"
  # (String) Token url
  authurl       = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"

  # (String) Client ID of the service account. It can be omitted if the environment variable AXUAL_CLIENT_ID is set.
  client_id     = "PLEASE_CHANGE_SERVICE_ACCOUNT_CLIENT_ID"
  # (String, Sensitive) Client secret of the service account. It can be omitted if the environment variable AXUAL_CLIENT_SECRET is set.
  client_secret = var.axual_client_secret
}

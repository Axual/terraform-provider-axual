# Legacy Axual Connect (AC) with a certificate, in the original flow:
# axual_application_deployment.autostart = true (the default).
#
# Order: principal -> grant + approval -> deployment. Creating the deployment starts the connector,
# so the principal and the approved grant must exist first.
#
# Replace every YOUR_* value below. This example uses its own directory, and therefore its own
# terraform.tfstate.

data "axual_instance" "example" {
  short_name = "YOUR_INSTANCE_SHORT_NAME" # short name of an existing instance
}

data "axual_group" "example" {
  name = "YOUR_GROUP_NAME" # an existing group; it owns everything below
}

resource "axual_environment" "example" {
  name                 = "tf-example-ac-tls-autostart"
  short_name           = "exactlsauto"
  description          = "AC+TLS connector with autostart"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.example.id
  owners               = data.axual_group.example.id
}

resource "axual_topic" "example" {
  name             = "tf-example-ac-tls-autostart"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.example.id
  retention_policy = "delete"
  properties       = {}
  description      = "Topic the AC+TLS connector reads"

  # A topic has no reference to the environment, so without this Terraform destroys both at the same
  # time, and their server-side cleanups collide on the same access grant.
  depends_on = [axual_environment.example]
}

resource "axual_topic_config" "example" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.example.id
  environment    = axual_environment.example.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "example" {
  name              = "tf-example-ac-tls-autostart"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "exactlsautoapp"
  application_id    = "tf.example.ac.tls.autostart"
  owners            = data.axual_group.example.id
  type              = "SINK"
  visibility        = "Public"
  description       = "AC+TLS connector with autostart"
}

# 1. The principal, active.
resource "axual_application_principal" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  principal   = file("certs/connector-cert.pem")        # the connector's certificate (chain), PEM
  private_key = file("certs/connector-private-key.key") # the connector's private key, PEM
  active      = true
}

# 2. Topic access. The grant depends on the principal, so a destroy revokes it before it deletes the
# principal.
resource "axual_application_access_grant" "example" {
  application = axual_application.example.id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.example, axual_application_principal.example]
}

resource "axual_application_access_grant_approval" "example" {
  application_access_grant = axual_application_access_grant.example.id
}

# 3. The deployment: no target_id (legacy Axual Connect). Creating it starts the connector.
resource "axual_application_deployment" "example" {
  application = axual_application.example.id
  environment = axual_environment.example.id
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
  depends_on = [axual_application_access_grant_approval.example]
}

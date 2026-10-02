# A registered Kafka Connect cluster (KC) that uses MTLS, with a certificate. Needs Platform Manager
# 16.0.0 or later. A KC connector only works with axual_application_deployment.autostart = false
# and an axual_application_deployment_state.
#
# Order: deployment -> principal -> grant + approval -> deployment state. The deployment comes
# first so the principal's private key goes straight to the Connect cluster's Vault.
#
# Replace every YOUR_* value below. This example uses its own directory, and therefore its own
# terraform.tfstate.

data "axual_instance" "example" {
  short_name = "YOUR_INSTANCE_SHORT_NAME" # short name of an existing instance
}

data "axual_group" "example" {
  name = "YOUR_GROUP_NAME" # an existing group; it owns everything below
}

data "axual_connect_cluster" "example" {
  instance_id = data.axual_instance.example.id
  cluster_id  = "YOUR_KAFKA_CLUSTER_ID"     # the Kafka cluster the Kafka Connect cluster is registered on
  name        = "YOUR_CONNECT_CLUSTER_NAME" # a Kafka Connect cluster with auth_method MTLS
}

resource "axual_environment" "example" {
  name                 = "tf-example-kc-tls"
  short_name           = "exkctls"
  description          = "KC+TLS connector"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.example.id
  owners               = data.axual_group.example.id
}

resource "axual_topic" "example" {
  name             = "tf-example-kc-tls"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.example.id
  retention_policy = "delete"
  properties       = {}
  description      = "Topic the KC+TLS connector reads"

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
  name              = "tf-example-kc-tls"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "exkctlsapp"
  application_id    = "tf.example.kc.tls"
  owners            = data.axual_group.example.id
  type              = "SINK"
  visibility        = "Public"
  description       = "KC+TLS connector"
}

# 1. The deployment on the Kafka Connect cluster. It stores the target and configs, and does not start.
resource "axual_application_deployment" "example" {
  application    = axual_application.example.id
  environment    = axual_environment.example.id
  target_id      = data.axual_connect_cluster.example.id
  target_version = "1.0.0" # the plugin version installed on the Kafka Connect cluster
  autostart      = false
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

# 2. The principal, active. It depends on the deployment, so its key goes to the Connect cluster's Vault.
resource "axual_application_principal" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  principal   = file("certs/connector-cert.pem")        # the connector's certificate (chain), PEM
  private_key = file("certs/connector-private-key.key") # the connector's private key, PEM
  active      = true
  depends_on  = [axual_application_deployment.example]
}

# 3. Topic access. The grant depends on the principal, so a destroy revokes it before it deletes the
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

# 4. Run it. To stop the connector, change state to "STOPPED" and apply. Removing this resource
# stops the connector and removes it from the Connect cluster.
resource "axual_application_deployment_state" "example" {
  application_deployment = axual_application_deployment.example.id
  state                  = "RUNNING"
  depends_on             = [axual_application_access_grant_approval.example, axual_application_principal.example]
}

output "current_state" {
  value = axual_application_deployment_state.example.current_state
}

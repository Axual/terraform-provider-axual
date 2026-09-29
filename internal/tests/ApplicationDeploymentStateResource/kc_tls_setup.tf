# KC+TLS in the Self-Service UI order: the deployment (target + configs, not started) first, then
# the principal, then the grant and its approval, then axual_application_deployment_state.

resource "axual_environment" "kc_tls" {
  name                 = "tf-ds-kc-tls"
  short_name           = "tfdskctls"
  description          = "axual_application_deployment_state KC+TLS test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_topic" "kc_tls" {
  name             = "tf-ds-kc-tls-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "axual_application_deployment_state KC+TLS test topic"
}

resource "axual_topic_config" "kc_tls" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.kc_tls.id
  environment    = axual_environment.kc_tls.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application" "kc_tls" {
  name              = "tf-test-app-ds-kc-tls"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdskctlsapp"
  application_id    = "tf.test.ds.kc.tls"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state KC+TLS test application"
}

data "axual_connect_cluster" "kc_tls" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
  id          = local.connect_cluster_mtls_id
}

# The deployment's target is known, so the private key goes straight to the Connect cluster's Vault.
resource "axual_application_principal" "kc_tls" {
  environment = axual_environment.kc_tls.id
  application = axual_application.kc_tls.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active      = true
  depends_on  = [axual_application_deployment.kc_tls]
}

# After the principal, so a destroy revokes the grant before it deletes the principal.
resource "axual_application_access_grant" "kc_tls" {
  application = axual_application.kc_tls.id
  topic       = axual_topic.kc_tls.id
  environment = axual_environment.kc_tls.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.kc_tls, axual_application_principal.kc_tls]
}

resource "axual_application_access_grant_approval" "kc_tls" {
  application_access_grant = axual_application_access_grant.kc_tls.id
}

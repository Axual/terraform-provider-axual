# autostart = true (the default) on an MTLS Kafka Connect cluster is refused too: the principal
# would exist before its target, so its key would not reach the Connect cluster's Vault.

resource "axual_environment" "tls_err" {
  name                 = "tf-ds-tls-err"
  short_name           = "tfdstlserr"
  description          = "axual_application_deployment autostart error test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "tls_err" {
  name              = "tf-test-app-ds-sasl-err"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdstlserrapp"
  application_id    = "tf.test.ds.tls.err"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment autostart error test application"
}

resource "axual_application_deployment" "tls_err" {
  application    = axual_application.tls_err.id
  environment    = axual_environment.tls_err.id
  target_id      = local.connect_cluster_mtls_id
  target_version = "1.0.0"
  configs = {
    "topics"   = "tf-ds-tls-err-topic",
    "endpoint" = "http://http-sink-target.kafka.svc.cluster.local:8080/",
  }
}

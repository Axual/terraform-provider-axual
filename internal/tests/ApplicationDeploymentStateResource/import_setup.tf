# A KC+TLS deployment and its active principal, to import back after Terraform forgot them.

resource "axual_environment" "imp" {
  name                 = "tf-ds-import"
  short_name           = "tfdsimport"
  description          = "axual_application_deployment_state import test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "imp" {
  name              = "tf-test-app-ds-import"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdsimportapp"
  application_id    = "tf.test.ds.import"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state import test application"
}

resource "axual_application_deployment" "imp" {
  application    = axual_application.imp.id
  environment    = axual_environment.imp.id
  target_id      = local.connect_cluster_mtls_id
  target_version = "1.0.0"
  autostart      = false
  configs = {
    "topics"   = "tf-ds-import-topic",
    "endpoint" = "http://http-sink-target.kafka.svc.cluster.local:8080/",
  }
}

resource "axual_application_principal" "imp" {
  environment = axual_environment.imp.id
  application = axual_application.imp.id
  principal   = file("{{CERTS}}/connector-cert.crt")
  private_key = file("{{CERTS}}/connector-cert.key")
  active      = true
  depends_on  = [axual_application_deployment.imp]
}

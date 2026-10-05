# autostart = true (the default) on a SASL_SCRAM Kafka Connect cluster cannot work: the credential
# it would need can only be created after the deployment exists. The provider says so up front.

resource "axual_environment" "sasl_err" {
  name                 = "tf-ds-sasl-err"
  short_name           = "tfdssaslerr"
  description          = "axual_application_deployment autostart error test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "sasl_err" {
  name              = "tf-test-app-ds-sasl-err"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdssaslerrapp"
  application_id    = "tf.test.ds.sasl.err"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment autostart error test application"
}

resource "axual_application_deployment" "sasl_err" {
  application    = axual_application.sasl_err.id
  environment    = axual_environment.sasl_err.id
  target_id      = local.connect_cluster_sasl_id
  target_version = "1.0.0"
  configs = {
    "topics"   = "tf-ds-sasl-err-topic",
    "endpoint" = "http://http-sink-target.kafka.svc.cluster.local:8080/",
  }
}

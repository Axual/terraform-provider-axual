# The environment and application for the plan-time autostart check: they exist before the
# deployment is planned, so the provider can read the application's type during plan.

resource "axual_environment" "tls_plan" {
  name                 = "tf-ds-tls-plan"
  short_name           = "tfdstlsplan"
  description          = "axual_application_deployment autostart error test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "tls_plan" {
  name              = "tf-test-app-ds-sasl-err"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdstlsplanapp"
  application_id    = "tf.test.ds.tls.plan"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment autostart error test application"
}

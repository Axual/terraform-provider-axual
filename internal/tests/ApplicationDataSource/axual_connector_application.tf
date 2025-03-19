resource "axual_application" "tf-test-app" {
  name              = "tf-test-app"
  application_type  = "Connector"
  application_class = "org.apache.kafka.connect.axual.utils.LogSourceConnector"
  short_name        = "tf_test_app"
  application_id    = "tf.test.app"
  owners            = data.axual_group.test_group.id
  type              = "SOURCE"
  visibility        = "Public"
  description       = "Axual's TF Test Application"
}

data "axual_application" "tf-test-app-imported" {
  name = axual_application.tf-test-app.name
}
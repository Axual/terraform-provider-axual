resource "axual_environment" "tf-test-env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the terraform testing environment"
  color                = "#19b9be"
  visibility           = "Private"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

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
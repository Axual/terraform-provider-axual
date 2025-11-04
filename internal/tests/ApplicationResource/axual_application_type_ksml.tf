resource "axual_application" "tf_test_app_type_ksml" {
  name             = "tf-test app type ksml"
  application_type = "Custom"
  short_name       = "tf_test_app_type_ksml"
  application_id   = "tf.test.app.type.ksml"
  owners           = data.axual_group.test_group.id
  type             = "KSML"
  visibility       = "Public"
  description      = "Test Application for KSML type"
}

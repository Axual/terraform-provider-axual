resource "axual_application" "tf_test_app_type_bridge" {
  name             = "tf-test app type bridge"
  application_type = "Custom"
  short_name       = "tf_test_app_type_bridge"
  application_id   = "tf.test.app.type.bridge"
  owners           = data.axual_group.test_group.id
  type             = "Bridge"
  visibility       = "Public"
  description      = "Test Application for Bridge type"
}

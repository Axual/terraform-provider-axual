resource "axual_application" "tf_test_app_type_python" {
  name             = "tf-test app type python"
  application_type = "Custom"
  short_name       = "tf_test_app_type_python"
  application_id   = "tf.test.app.type.python"
  owners           = data.axual_group.test_group.id
  type             = "Python"
  visibility       = "Public"
  description      = "Test Application for Python type"
}

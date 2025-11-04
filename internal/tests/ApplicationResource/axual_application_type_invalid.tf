resource "axual_application" "tf-test-app-invalid" {
  name             = "tf-test app type invalid"
  application_type = "Custom"
  short_name       = "tf_test_app_type_invalid"
  application_id   = "tf.test.app.type.invalid"
  owners           = data.axual_group.test_group.id
  type             = "InvalidType"
  visibility       = "Public"
  description      = "Test Application with invalid type - should fail"
}

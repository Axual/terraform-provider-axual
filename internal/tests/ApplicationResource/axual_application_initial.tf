resource "axual_application" "tf-test-app" {
  name             = "tf-test app"
  application_type = "Custom"
  short_name       = "tf_test_app"
  application_id   = "tf.test.app"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Axual's TF Test Application"
}
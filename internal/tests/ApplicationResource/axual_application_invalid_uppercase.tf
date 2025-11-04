resource "axual_application" "tf_test_app" {
  name             = "tf-test app"
  application_type = "Custom"
  short_name       = "TF_Test_App" # Mixed case - should fail
  application_id   = "tf.test.app"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Axual's TF Test Application"
}

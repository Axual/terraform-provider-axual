resource "axual_application" "tf-test-app" {
  name             = "tf-test-app1"
  application_type = "Custom"
  short_name       = "tf_test_app1"
  application_id   = "tf.test.app1"
  owners           = data.axual_group.test_group.id
  type             = "Pega"
  visibility       = "Private"
  description      = "Axual's TF Test Application1"
}
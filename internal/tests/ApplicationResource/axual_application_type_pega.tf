resource "axual_application" "tf_test_app_type_pega" {
  name             = "tf-test app type pega"
  application_type = "Custom"
  short_name       = "tf_test_app_type_pega"
  application_id   = "tf.test.app.type.pega"
  owners           = data.axual_group.test_group.id
  type             = "Pega"
  visibility       = "Public"
  description      = "Test Application for Pega type"
}

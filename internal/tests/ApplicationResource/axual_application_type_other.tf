resource "axual_application" "tf_test_app_type_other" {
  name             = "tf-test app type other"
  application_type = "Custom"
  short_name       = "tf_test_app_type_other"
  application_id   = "tf.test.app.type.other"
  owners           = data.axual_group.test_group.id
  type             = "Other"
  visibility       = "Public"
  description      = "Test Application for Other type"
}

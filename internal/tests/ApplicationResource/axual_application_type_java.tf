resource "axual_application" "tf_test_app_type_java" {
  name             = "tf-test app type java"
  application_type = "Custom"
  short_name       = "tf_test_app_type_java"
  application_id   = "tf.test.app.type.java"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Test Application for Java type"
}

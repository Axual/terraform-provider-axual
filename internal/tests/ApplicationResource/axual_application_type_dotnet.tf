resource "axual_application" "tf_test_app_type_dotnet" {
  name             = "tf-test app type dotnet"
  application_type = "Custom"
  short_name       = "tf_test_app_type_dotnet"
  application_id   = "tf.test.app.type.dotnet"
  owners           = data.axual_group.test_group.id
  type             = "DotNet"
  visibility       = "Public"
  description      = "Test Application for DotNet type"
}

# resource "axual_application" "tf-test-app" {
#   name             = "tf-test-app"
#   application_type = "Custom"
#   short_name       = "tf_test_app_short"
#   application_id   = "tf.test.app"
#   owners           = data.axual_group.test_group.id
#   type             = "Java"
#   visibility       = "Public"
#   description      = "Axual's TF Test Application"
# }

data "axual_application" "tf-test-app-imported-by-name" {
  name = "tf-test-app"
}

data "axual_application" "tf-test-app-imported-by-short-name" {
  short_name = "tf_test_app_short"
}

data "axual_application" "tf-test-app-imported-by-short-name-and-name" {
  name = "tf-test-app"
  short_name = "tf_test_app_short"
}
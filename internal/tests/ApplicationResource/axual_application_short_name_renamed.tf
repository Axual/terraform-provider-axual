resource "axual_application" "tf-test-app" {
  name             = "tf-test-app-short-name"
  application_type = "Custom"
  short_name       = "tf_test_app_short_name1"
  application_id   = "tf.test.app.short.name"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/generic_application_1.cer")

  lifecycle {
    replace_triggered_by = [axual_application.tf-test-app.short_name]
  }
}

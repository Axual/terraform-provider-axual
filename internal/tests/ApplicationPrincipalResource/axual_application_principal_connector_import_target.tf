resource "axual_application_principal" "import_target" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/generic_application_1.cer")
  private_key = file("{{CERTS}}/generic_application_1.key")
}

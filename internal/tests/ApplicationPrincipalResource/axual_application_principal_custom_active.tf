resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/generic_application_3.cer")
  active      = true
}

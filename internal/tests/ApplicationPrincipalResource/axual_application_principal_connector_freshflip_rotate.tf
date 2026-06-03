resource "axual_application_principal" "connector_axual_application_principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/generic_application_2.cer")
  private_key = file("{{CERTS}}/generic_application_2.key")
  active      = true
}

resource "axual_application_principal" "connector_axual_application_principal_other" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("{{CERTS}}/generic_application_4.cer")
  private_key = file("{{CERTS}}/generic_application_4.key")
}

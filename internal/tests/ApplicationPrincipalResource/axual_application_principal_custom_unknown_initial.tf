# The certificate goes through another resource, so a new one is only known at apply, like a
# certificate from a module output or a tls_* resource (GitHub #175).
resource "terraform_data" "certificate" {
  input = file("{{CERTS}}/generic_application_3.cer")
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = terraform_data.certificate.output
}

# terraform_data is replaced, so the certificate is unknown at plan, but it is the same certificate.
resource "terraform_data" "certificate" {
  input            = file("{{CERTS}}/example_stream_processor.cer")
  triggers_replace = "replace-without-a-new-certificate"
}

resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = terraform_data.certificate.output
}

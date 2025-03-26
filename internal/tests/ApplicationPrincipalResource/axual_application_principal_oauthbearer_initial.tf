resource "axual_application_principal" "tf-test-app-principal" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = "example-oauthbearer-principal"
  custom      = true
}
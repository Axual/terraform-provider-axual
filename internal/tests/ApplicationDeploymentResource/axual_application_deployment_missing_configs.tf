resource "axual_application_deployment" "connector_axual_application_deployment" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  depends_on = [
    axual_application_access_grant_approval.tf-test-application-access-grant-approval,
  ]
}

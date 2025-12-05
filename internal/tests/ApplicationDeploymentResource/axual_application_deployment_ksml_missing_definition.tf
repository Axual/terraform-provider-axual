resource "axual_application_deployment" "ksml_axual_application_deployment" {
  environment     = axual_environment.tf-test-ksml-env.id
  application     = axual_application.tf-test-ksml-app.id
  deployment_size = "M"
  restart_policy  = "on_exit"
  depends_on = [
    axual_application_access_grant_approval.tf-test-ksml-application-access-grant-approval,
  ]
}

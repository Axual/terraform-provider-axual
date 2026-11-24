# A second KSML runtime with the same URL: the deployment moves to it in place.
resource "axual_ksml_runtime" "tf_test_ksml_runtime_2" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ksml_runtime_url
  name              = "tf-test-ksml-deployment-runtime-2"
  owners            = data.axual_group.test_group.id
  authorized_groups = [data.axual_group.test_group.id]
}

resource "axual_application_deployment" "ksml_axual_application_deployment" {
  environment     = axual_environment.tf-test-ksml-env.id
  application     = axual_application.tf-test-ksml-app.id
  target_id       = axual_ksml_runtime.tf_test_ksml_runtime_2.id
  definition      = file("definitions/ksml-definition.yaml")
  deployment_size = "S"
  restart_policy  = "on_exit"
  depends_on = [
    axual_application_access_grant_decision.tf-test-ksml-application-access-grant-approval,
  ]
}

# A KSML runtime without authorized Groups accepts no deployment, from its owner Group either.
resource "axual_ksml_runtime" "tf_test_ksml_runtime_closed" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ksml_runtime_url
  name              = "tf-test-ksml-deployment-runtime-closed"
  owners            = data.axual_group.test_group.id
  authorized_groups = []
}

resource "axual_application_deployment" "ksml_axual_application_deployment" {
  environment     = axual_environment.tf-test-ksml-env.id
  application     = axual_application.tf-test-ksml-app.id
  target_id       = axual_ksml_runtime.tf_test_ksml_runtime_closed.id
  definition      = file("definitions/ksml-definition.yaml")
  deployment_size = "S"
  restart_policy  = "on_exit"
  depends_on = [
    axual_application_access_grant_decision.tf-test-ksml-application-access-grant-approval,
  ]
}

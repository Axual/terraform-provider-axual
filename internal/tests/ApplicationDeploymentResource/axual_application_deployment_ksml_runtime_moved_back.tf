# The deployment is back on the first KSML runtime. The second one stays declared: Terraform deletes a
# resource that left the configuration before it updates what depended on it, and Platform Manager
# refuses to delete a KSML runtime that a deployment still targets.
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
  target_id       = axual_ksml_runtime.tf_test_ksml_runtime.id
  definition      = file("definitions/ksml-definition.yaml")
  deployment_size = "S"
  restart_policy  = "on_exit"
  depends_on = [
    axual_application_access_grant_decision.tf-test-ksml-application-access-grant-approval,
  ]
}

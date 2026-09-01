resource "axual_application_deployment" "flink_axual_application_deployment" {
  environment = axual_environment.tf-test-flink-env.id
  application = axual_application.tf-test-flink-app.id
  target_id   = axual_flink_cluster.tf-test-flink-cluster.id
  deployment_size = "S"
  depends_on = [
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval,
  ]
}

resource "axual_application_deployment" "flink_axual_application_deployment" {
  environment         = axual_environment.tf-test-flink-env.id
  application         = axual_application.tf-test-flink-app.id
  target_id           = axual_flink_cluster.tf-test-flink-cluster.id
  sql_script          = "INSERT INTO flink_test_topic SELECT id, value FROM flink_test_topic"
  generate_tables_sql = true
  task_size           = "M"
  depends_on = [
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval,
  ]
}

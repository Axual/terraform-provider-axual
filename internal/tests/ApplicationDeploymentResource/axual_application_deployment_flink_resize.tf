resource "axual_application_deployment" "flink_axual_application_deployment" {
  environment         = axual_environment.tf-test-flink-env.id
  application         = axual_application.tf-test-flink-app.id
  target_id           = axual_flink_cluster.tf-test-flink-cluster.id
  deployment_size     = "M"
  generate_tables_sql = true

  sql_script = <<-SQL
    INSERT INTO flink_test_topic_out (`key`, `value`)
    SELECT `key`, UPPER(`value`) FROM flink_test_topic;
  SQL

  depends_on = [
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval,
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval-out,
  ]
}

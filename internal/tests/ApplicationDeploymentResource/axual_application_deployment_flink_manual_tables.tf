# `generate_tables_sql` is left at its default (false), so the script declares the tables itself,
# mirroring what FlinkTableDdlGenerator produces. The Kafka connection options are injected by the
# platform and must not appear here; `topic` is the resolved name, built from `resolvedTopicPrefix`.
resource "axual_application_deployment" "flink_axual_application_deployment" {
  environment     = axual_environment.tf-test-flink-env.id
  application     = axual_application.tf-test-flink-app.id
  target_id       = axual_flink_cluster.tf-test-flink-cluster.id
  deployment_size = "S"

  sql_script = <<-SQL
    CREATE TEMPORARY TABLE `flink_test_topic` (
      `key` STRING,
      `value` STRING
    ) WITH (
      'connector' = 'kafka',
      'topic' = '${local.resolved_topic_prefix}flink-test-topic',
      'scan.startup.mode' = 'earliest-offset',
      'key.format' = 'raw',
      'key.fields' = 'key',
      'value.format' = 'raw',
      'value.fields-include' = 'EXCEPT_KEY'
    );

    CREATE TEMPORARY TABLE `flink_test_topic_out` (
      `key` STRING,
      `value` STRING
    ) WITH (
      'connector' = 'kafka',
      'topic' = '${local.resolved_topic_prefix}flink-test-topic-out',
      'scan.startup.mode' = 'earliest-offset',
      'key.format' = 'raw',
      'key.fields' = 'key',
      'value.format' = 'raw',
      'value.fields-include' = 'EXCEPT_KEY'
    );

    INSERT INTO flink_test_topic_out (`key`, `value`)
    SELECT `key`, `value` FROM flink_test_topic;
  SQL

  depends_on = [
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval,
    axual_application_access_grant_approval.tf-test-flink-application-access-grant-approval-out,
  ]
}

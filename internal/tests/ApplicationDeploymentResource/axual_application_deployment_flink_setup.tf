resource "axual_application" "tf-test-flink-app" {
  name             = "tf-test-flink-app"
  application_type = "FLINK_SQL"
  short_name       = "tf_test_flink_app"
  application_id   = "tf.test.flink.app"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  description      = "Axual's TF Test FLINK_SQL Application"
}

resource "axual_environment" "tf-test-flink-env" {
  name                 = "tf-flink-development"
  short_name           = "tfflinkdev"
  description          = "This is the Flink development environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_flink_cluster" "tf-test-flink-cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  name              = "tf-test-flink-deployment-cluster"
  description       = "Axual's TF Test Flink Cluster for deployments"
  url               = local.ververica_url
  workspace         = local.ververica_workspace
  namespace         = local.ververica_namespace
  deployment_target = local.ververica_deployment_target
  api_token         = local.ververica_api_token
}

# A FLINK_SQL application needs SASL/SCRAM Kafka credentials, not an mTLS principal: the platform
# rejects the deployment with "a Flink SQL application requires SASL/SCRAM Kafka credentials. Only
# an mTLS principal is configured, which Flink SQL cannot use". The credential also satisfies the
# Application Access Grant endpoint, which refuses a grant for an application without any
# authentication information.
resource "axual_application_credential" "flink_axual_application_credential" {
  environment = axual_environment.tf-test-flink-env.id
  application = axual_application.tf-test-flink-app.id
  target      = "KAFKA"
}

resource "axual_topic" "tf-test-flink-topic" {
  name             = "flink-test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Demo of Flink SQL topic via Terraform"

  # A topic has no reference to an environment, so Terraform destroys both in parallel and their
  # server-side cascades collide on the same access-grant row. This edge serialises them.
  depends_on = [axual_environment.tf-test-flink-env]
}

resource "axual_topic_config" "tf-flink-topic-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-flink-topic.id
  environment    = axual_environment.tf-test-flink-env.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_application_access_grant" "tf-test-flink-application-access-grant" {
  application = axual_application.tf-test-flink-app.id
  topic       = axual_topic.tf-test-flink-topic.id
  environment = axual_environment.tf-test-flink-env.id
  access_type = "CONSUMER"
  depends_on = [
    axual_application_credential.flink_axual_application_credential,
    axual_topic_config.tf-flink-topic-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-flink-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-flink-application-access-grant.id
}

resource "axual_topic" "tf-test-flink-topic-out" {
  name             = "flink-test-topic-out"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Target topic of the Flink SQL job created via Terraform"

  # Same reason as the source topic above: destroy the topic before the environment.
  depends_on = [axual_environment.tf-test-flink-env]
}

resource "axual_topic_config" "tf-flink-topic-out-config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.tf-test-flink-topic-out.id
  environment    = axual_environment.tf-test-flink-env.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

# The job reads from the source topic and writes to this one, so it needs both a CONSUMER and a
# PRODUCER grant: with generate_tables_sql = true the platform generates the table DDL from the
# approved grants, so a topic without a grant has no table to select from or insert into.
resource "axual_application_access_grant" "tf-test-flink-application-access-grant-out" {
  application = axual_application.tf-test-flink-app.id
  topic       = axual_topic.tf-test-flink-topic-out.id
  environment = axual_environment.tf-test-flink-env.id
  access_type = "PRODUCER"
  depends_on = [
    axual_application_credential.flink_axual_application_credential,
    axual_topic_config.tf-flink-topic-out-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-flink-application-access-grant-approval-out" {
  application_access_grant = axual_application_access_grant.tf-test-flink-application-access-grant-out.id
}

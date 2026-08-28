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
  cluster_id        = local.instance_cluster_id
  name              = "tf-test-flink-deployment-cluster"
  description       = "Axual's TF Test Flink Cluster for deployments"
  url               = "https://vvp.example.internal/api"
  workspace         = "defaultworkspace"
  namespace         = "default"
  deployment_target = "default-target"
  api_token         = "tf-test-flink-deployment-token"
}

resource "axual_topic" "tf-test-flink-topic" {
  name             = "flink-test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Demo of Flink SQL topic via Terraform"
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
    axual_topic_config.tf-flink-topic-config
  ]
}

resource "axual_application_access_grant_approval" "tf-test-flink-application-access-grant-approval" {
  application_access_grant = axual_application_access_grant.tf-test-flink-application-access-grant.id
}

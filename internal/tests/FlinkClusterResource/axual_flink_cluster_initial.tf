resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ververica_url
  api_token         = local.ververica_api_token
  name              = "tf-test-flink-cluster"
  owners            = data.axual_group.test_group.id
  authorized_groups = [data.axual_group.test_group.id]
  description       = "Axual's TF Test Flink Cluster"
  workspace         = local.ververica_workspace
  namespace         = local.ververica_namespace
  deployment_target = local.ververica_deployment_target
}

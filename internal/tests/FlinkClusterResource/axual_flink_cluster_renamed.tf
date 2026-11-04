# Platform Manager does not allow renaming a Flink Cluster, so a changed `name` replaces the resource:
# the old Flink Cluster is deleted and a new one is registered under the new name.
resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ververica_url
  api_token         = local.ververica_api_token
  name              = "tf-test-flink-cluster-renamed"
  owner             = data.axual_group.test_group.id
  authorized_groups = [data.axual_group.test_group.id]
  workspace         = local.ververica_workspace
  namespace         = local.ververica_namespace
  deployment_target = local.ververica_deployment_target
}

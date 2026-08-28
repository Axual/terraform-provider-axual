resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.instance_cluster_id
  name              = "tf-test-flink-cluster"
  description       = "Axual's TF Test Flink Cluster, updated"
  url               = "https://vvp.example.internal/api"
  workspace         = "defaultworkspace"
  namespace         = "default"
  deployment_target = "default-target"
  api_token         = "tf-test-updated-token"
}

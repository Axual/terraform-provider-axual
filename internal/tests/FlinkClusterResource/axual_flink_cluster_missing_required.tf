resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id = data.axual_instance.test_instance.id
  cluster_id  = local.instance_cluster_id
  name        = "tf-test-flink-cluster"
}

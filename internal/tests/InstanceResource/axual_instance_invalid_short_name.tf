resource "axual_instance" "tf_test_instance" {
  name       = "tf-test-instance"
  short_name = "TfTestInst"
  clusters   = [axual_cluster.tf_test_instance_cluster_a.id]
}

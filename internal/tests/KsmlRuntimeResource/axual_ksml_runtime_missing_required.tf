resource "axual_ksml_runtime" "tf_test_ksml_runtime" {
  instance_id = data.axual_instance.test_instance.id
  cluster_id  = local.cluster_id
  name        = "tf-test-ksml-runtime"
}

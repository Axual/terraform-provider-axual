resource "axual_ksml_runtime" "tf_test_ksml_runtime" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ksml_runtime_url
  name              = "tf-test-ksml-runtime"
  owners            = data.axual_group.test_group.id
  authorized_groups = [data.axual_group.test_group.id]
  description       = "Axual's TF Test KSML runtime"
}

# `description` and `schema_registries` are left out: the PATCH has to carry an explicit null for
# each to clear what is stored, or the API keeps it and the apply fails with
# "Provider produced inconsistent result after apply".
resource "axual_ksml_runtime" "tf_test_ksml_runtime" {
  instance_id = data.axual_instance.test_instance.id
  cluster_id  = local.cluster_id
  url         = local.ksml_runtime_url
  name        = "tf-test-ksml-runtime"
  owners      = data.axual_group.test_group.id
}

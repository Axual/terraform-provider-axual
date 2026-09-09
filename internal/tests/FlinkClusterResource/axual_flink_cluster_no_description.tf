# `description` and `schema_registries` are left out and `name` is changed: the PATCH has to carry an explicit
# "description": null to clear the stored text, or the API keeps it and the apply fails with
# "Provider produced inconsistent result after apply".
resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ververica_url
  api_token         = local.ververica_api_token
  name              = "tf-test-flink-cluster-renamed"
  workspace         = "defaultworkspace"
  namespace         = "default"
  deployment_target = "default-target"
}

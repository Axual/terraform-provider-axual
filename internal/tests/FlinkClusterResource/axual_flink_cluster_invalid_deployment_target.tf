# `deployment_target` is changed to a name Ververica does not know: create and update both validate
# the target live against the Ververica namespace, so the API rejects the PATCH. There is only one
# real deployment target on the test Ververica, so this is what covers a target change.
resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ververica_url
  api_token         = local.ververica_api_token
  name              = "tf-test-flink-cluster-renamed"
  workspace         = local.ververica_workspace
  namespace         = local.ververica_namespace
  deployment_target = "tf-test-target-does-not-exist"
}

resource "axual_flink_cluster" "tf_test_flink_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = local.cluster_id
  url               = local.ververica_url
  api_token         = local.ververica_api_token
  name              = "tf-test-flink-cluster"
  description       = "Axual's TF Test Flink Cluster, updated"
  workspace         = "defaultworkspace"
  namespace         = "default"
  deployment_target = "default-target"

  # Added on update: a Flink SQL job over AVRO topics needs a schema registry, and the next step
  # removes the block again, which the PATCH can only express as an explicit null.
  schema_registries = [
    {
      type = "CONFLUENT"
      url  = "https://schema-registry.example.com/apis/ccompat/v7"
    },
  ]
}

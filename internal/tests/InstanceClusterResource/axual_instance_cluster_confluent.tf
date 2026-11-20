resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "CONFLUENT"
    url            = replace(local.schema_registry_url, "/apis/registry/v3", "/apis/ccompat/v7")
    authentication = "No-Auth"
    version        = "7.0"
  }
}
